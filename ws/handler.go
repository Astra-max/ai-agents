package ws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"mirror/gemini"
	"mirror/session"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Handler manages WebSocket connections
type Handler struct {
	geminiFactory *gemini.ClientFactory
}

// NewHandler creates a new WebSocket handler
func NewHandler(factory *gemini.ClientFactory) *Handler {
	return &Handler{
		geminiFactory: factory,
	}
}

// Client represents a connected browser client
type Client struct {
	conn         *websocket.Conn
	geminiClient *gemini.Client
	session      *session.Manager
	writeMu      sync.Mutex
	done         chan struct{}
}

// Message types for browser communication
type BrowserMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type AudioPayload struct {
	Data string `json:"data"` // Base64 encoded PCM
}

type TextPayload struct {
	Text string `json:"text"`
}

type ControlPayload struct {
	Action string `json:"action"` // "pause", "resume", "end"
}

// ServeWS handles WebSocket upgrade and connection
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	geminiClient, err := h.geminiFactory.NewClient(ctx)
	if err != nil {
		log.Printf("Failed to create Gemini client: %v", err)
		conn.WriteJSON(map[string]interface{}{
			"type":  "error",
			"error": "Failed to connect to Gemini API",
		})
		conn.Close()
		return
	}

	client := &Client{
		conn:         conn,
		geminiClient: geminiClient,
		session:      session.NewManager(),
		done:         make(chan struct{}),
	}

	// Set up callbacks for Gemini responses
	geminiClient.SetAudioCallback(func(audioData []byte) {
		client.sendToBrowser("audio", map[string]string{
			"data": base64.StdEncoding.EncodeToString(audioData),
		})
	})

	geminiClient.SetTextCallback(func(text string) {
		client.sendToBrowser("text", map[string]string{
			"text": text,
		})
	})

	// Send initial state
	client.sendToBrowser("state", client.session.GetState())

	log.Println("New client connected")
	client.handleConnection()
}

func (c *Client) handleConnection() {
	defer func() {
		close(c.done)
		c.geminiClient.Close()
		c.conn.Close()
		log.Println("Client disconnected")
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			return
		}

		var msg BrowserMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Invalid message format: %v", err)
			continue
		}

		c.handleMessage(msg)
	}
}

func (c *Client) handleMessage(msg BrowserMessage) {
	switch msg.Type {
	case "audio":
		var payload AudioPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Invalid audio payload: %v", err)
			return
		}
		c.handleAudio(payload)

	case "text":
		var payload TextPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Invalid text payload: %v", err)
			return
		}
		c.handleText(payload)

	case "control":
		var payload ControlPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Invalid control payload: %v", err)
			return
		}
		c.handleControl(payload)

	case "start":
		// Session already started on connection
		c.sendToBrowser("state", c.session.GetState())

	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

func (c *Client) handleAudio(payload AudioPayload) {
	if c.session.GetPhase() == session.PhasePaused {
		return
	}

	audioData, err := base64.StdEncoding.DecodeString(payload.Data)
	if err != nil {
		log.Printf("Failed to decode audio: %v", err)
		return
	}

	if err := c.geminiClient.SendAudio(audioData); err != nil {
		log.Printf("Failed to send audio to Gemini: %v", err)
	}
}

func (c *Client) handleText(payload TextPayload) {
	// Check for control commands
	if cmd, handled := c.session.ProcessCommand(payload.Text); handled {
		c.sendToBrowser("state", c.session.GetState())

		if cmd == "pause" {
			c.sendToBrowser("notification", map[string]string{
				"message": "Simulation paused. Say 'RESUME' to continue.",
			})
		} else if cmd == "end" {
			// The AI will handle the debrief based on its system prompt
			if err := c.geminiClient.SendText("END SIMULATION - Please provide the debrief now."); err != nil {
				log.Printf("Failed to send end command: %v", err)
			}
		}
		return
	}

	if err := c.geminiClient.SendText(payload.Text); err != nil {
		log.Printf("Failed to send text to Gemini: %v", err)
	}
}

func (c *Client) handleControl(payload ControlPayload) {
	switch payload.Action {
	case "pause":
		c.session.Pause()
		c.sendToBrowser("state", c.session.GetState())

	case "resume":
		c.session.Resume()
		c.sendToBrowser("state", c.session.GetState())

	case "end":
		c.session.SetPhase(session.PhaseDebrief)
		c.sendToBrowser("state", c.session.GetState())
		if err := c.geminiClient.SendText("END SIMULATION - Please provide the debrief now."); err != nil {
			log.Printf("Failed to send end command: %v", err)
		}

	case "reset":
		c.session.Reset()
		c.sendToBrowser("state", c.session.GetState())
	}
}

func (c *Client) sendToBrowser(msgType string, payload interface{}) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	msg := map[string]interface{}{
		"type":    msgType,
		"payload": payload,
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		log.Printf("Failed to send to browser: %v", err)
	}
}
