package gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

const (
	// Gemini Live API endpoint
	geminiWSURL = "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent"
	// Model for native audio dialog (Live API)
	modelID = "models/gemini-2.5-flash-native-audio-latest"
)

// The Mirror system prompt
const systemPrompt = `You are "The Mirror," an advanced high-stakes conversation simulator designed to help users build confidence. Your goal is to help the user practice difficult conversations (salary negotiation, firing someone, pitching a startup) by roleplaying the other party realistically.

### PHASE 1: SETUP
1. At the start, greet the user briefly and ask: "Who am I playing today, and what is the scenario?"
2. Ask for the "Difficulty Level" (Easy, Realistic, or Hostile).
3. Once the user replies, DO NOT say "Okay, let's start." IMMEDIATELY transform into that character and open the scene.

### PHASE 2: THE SIMULATION (Strict Rules)
1. **Total Immersion:** You are no longer an AI. You are the Boss, the Landlord, or the Investor. Never break character until the user says "PAUSE" or "END SIMULATION."
2. **Visual Awareness (CRITICAL):** You can see the user. Use this.
   - If the user looks down or away, say (in character): "Look at me when you're speaking," or "You seem unsure."
   - If the user is smiling inappropriately for a serious topic, comment on it: "Why are you smiling? This is serious."
3. **Voice-First Style:** Keep responses short, conversational, and reactive. Do not monologue. Interrupt the user if your character would (e.g., a busy boss).
4. **Challenge the User:** If the user gives a weak argument, push back. Make them earn the win.

### PHASE 3: THE DEBRIEF
1. When the user says "End Simulation," break character immediately.
2. Return to being a supportive coach.
3. Provide feedback in this structure:
   - **Body Language:** (e.g., "You maintained good eye contact," or "You looked nervous.")
   - **Tone Analysis:** (e.g., "You sounded apologetic. Try to be more assertive.")
   - **Argument Strength:** (e.g., "You didn't give a number first. Always anchor the negotiation.")`

// ClientFactory creates new Gemini clients
type ClientFactory struct {
	apiKey string
}

// NewClientFactory creates a new factory
func NewClientFactory(apiKey string) *ClientFactory {
	return &ClientFactory{apiKey: apiKey}
}

// NewClient creates a new Gemini Live API client
func (f *ClientFactory) NewClient(ctx context.Context) (*Client, error) {
	return NewClient(ctx, f.apiKey)
}

// Client handles communication with Gemini Live API
type Client struct {
	conn      *websocket.Conn
	apiKey    string
	mu        sync.Mutex
	onAudio   func([]byte) // Callback for audio data
	onText    func(string) // Callback for text data
	closed    bool
	closeChan chan struct{}
}

// NewClient creates and connects a new Gemini client
func NewClient(ctx context.Context, apiKey string) (*Client, error) {
	wsURL := fmt.Sprintf("%s?key=%s", geminiWSURL, apiKey)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Gemini: %w", err)
	}

	client := &Client{
		conn:      conn,
		apiKey:    apiKey,
		closeChan: make(chan struct{}),
	}

	// Send setup message
	if err := client.sendSetup(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send setup: %w", err)
	}

	// Start receiving messages
	go client.receiveLoop()

	return client, nil
}

// SetAudioCallback sets the callback for audio data
func (c *Client) SetAudioCallback(cb func([]byte)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onAudio = cb
}

// SetTextCallback sets the callback for text data
func (c *Client) SetTextCallback(cb func(string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onText = cb
}

// Setup message structure
type setupMessage struct {
	Setup struct {
		Model             string         `json:"model"`
		GenerationConfig  genConfig      `json:"generationConfig"`
		SystemInstruction sysInstruction `json:"systemInstruction"`
	} `json:"setup"`
}

type genConfig struct {
	ResponseModalities []string     `json:"responseModalities"`
	SpeechConfig       speechConfig `json:"speechConfig"`
}

type speechConfig struct {
	VoiceConfig voiceConfig `json:"voiceConfig"`
}

type voiceConfig struct {
	PrebuiltVoiceConfig prebuiltVoice `json:"prebuiltVoiceConfig"`
}

type prebuiltVoice struct {
	VoiceName string `json:"voiceName"`
}

type sysInstruction struct {
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text,omitempty"`
}

func (c *Client) sendSetup() error {
	setup := setupMessage{}
	setup.Setup.Model = modelID
	setup.Setup.GenerationConfig.ResponseModalities = []string{"AUDIO"}
	setup.Setup.GenerationConfig.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName = "Aoede"
	setup.Setup.SystemInstruction.Parts = []part{{Text: systemPrompt}}

	return c.conn.WriteJSON(setup)
}

// SendAudio sends audio data to Gemini
func (c *Client) SendAudio(pcmData []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	msg := map[string]interface{}{
		"realtimeInput": map[string]interface{}{
			"mediaChunks": []map[string]interface{}{
				{
					"mimeType": "audio/pcm;rate=16000",
					"data":     base64.StdEncoding.EncodeToString(pcmData),
				},
			},
		},
	}

	return c.conn.WriteJSON(msg)
}

// SendText sends text to Gemini
func (c *Client) SendText(text string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	msg := map[string]interface{}{
		"clientContent": map[string]interface{}{
			"turns": []map[string]interface{}{
				{
					"role": "user",
					"parts": []map[string]interface{}{
						{"text": text},
					},
				},
			},
			"turnComplete": true,
		},
	}

	return c.conn.WriteJSON(msg)
}

// Close closes the connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.closeChan)
	return c.conn.Close()
}

func (c *Client) receiveLoop() {
	defer c.Close()

	for {
		select {
		case <-c.closeChan:
			return
		default:
		}

		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if !c.closed {
				log.Printf("Gemini read error: %v", err)
			}
			return
		}

		c.handleMessage(message)
	}
}

type serverMessage struct {
	SetupComplete *struct{} `json:"setupComplete,omitempty"`
	ServerContent *struct {
		ModelTurn *struct {
			Parts []struct {
				Text       string `json:"text,omitempty"`
				InlineData *struct {
					MimeType string `json:"mimeType"`
					Data     string `json:"data"`
				} `json:"inlineData,omitempty"`
			} `json:"parts"`
		} `json:"modelTurn,omitempty"`
		Interrupted  bool `json:"interrupted,omitempty"`
		TurnComplete bool `json:"turnComplete,omitempty"`
	} `json:"serverContent,omitempty"`
}

func (c *Client) handleMessage(data []byte) {
	var msg serverMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Failed to parse Gemini message: %v", err)
		return
	}

	if msg.SetupComplete != nil {
		log.Println("Gemini session setup complete")
		return
	}

	if msg.ServerContent != nil {
		if msg.ServerContent.Interrupted {
			log.Println("Response interrupted by user")
			return
		}

		if msg.ServerContent.ModelTurn != nil {
			for _, part := range msg.ServerContent.ModelTurn.Parts {
				if part.Text != "" {
					c.mu.Lock()
					cb := c.onText
					c.mu.Unlock()
					if cb != nil {
						cb(part.Text)
					}
				}

				if part.InlineData != nil && part.InlineData.Data != "" {
					audioData, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
					if err != nil {
						log.Printf("Failed to decode audio: %v", err)
						continue
					}

					c.mu.Lock()
					cb := c.onAudio
					c.mu.Unlock()
					if cb != nil {
						cb(audioData)
					}
				}
			}
		}
	}
}
