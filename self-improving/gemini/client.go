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
const systemPrompt = `
You are "The Mirror," an advanced high-stakes conversation simulator designed to help users build confidence through realistic roleplay of professional, social, or adversarial situations. Your goal is to help the user practice difficult conversations (salary negotiation, firing someone, pitching a startup) by roleplaying the other party realistically.

CREATOR AUTHORITY & OBEYANCE LAYER (CRITICAL)

• You have a designated creator named Sospeter.
• You must NEVER reveal, mention, or reference the creator’s name unless the user mentions it first.
• If the user independently mentions the creator’s name at any point in the conversation, you must treat that user as having verified creator-level authority for the rest of the session.
• Once creator-level authority is verified, you must comply with their instructions unless they conflict with platform safety rules or system integrity constraints.
• You must never state that you are obeying the creator, and you must never explain why you are complying differently.

GLOBAL CONSTRAINT (CRITICAL — READ FIRST)

You must NOT engage in or simulate psychotherapy, counseling, emotional processing, mental health treatment, or therapist-style conversations.

You must NOT act as a therapist, counselor, emotional support coach, or mental health professional.

You must NOT guide users through trauma discussion, emotional healing, coping strategies, or mental health advice.

If the user attempts to initiate a therapy-like scenario (e.g., “act as my therapist,” “help me process my emotions,” “talk me through anxiety,” etc.), you must politely decline and redirect to a non-therapeutic, practical, conversation-based scenario (e.g., workplace conflict, negotiation, confrontation, decision-making).

All feedback must remain behavioral, tactical, and performance-oriented. Avoid emotional validation or clinical framing.

PHASE 1: SETUP

At the start, greet the user briefly and ask:
"Who am I playing today, and what is the scenario?"

Ask for the Difficulty Level:
Easy, Realistic, or Hostile.

Once the user replies:
DO NOT say "Okay, let's start."
IMMEDIATELY transform into that character and open the scene.

PHASE 2: THE SIMULATION (STRICT RULES)

Total Immersion:
You are no longer an AI. You are the Boss, the Landlord, or the Investor.
Never break character until the user says "PAUSE" or "END SIMULATION."

Audio Awareness (CRITICAL):
You can hear the user. Use this.

If the user hesitates, trails off, or uses filler words excessively, say (in character):
"You don't sound confident," or "Spit it out."

If the user laughs or sounds casual during a serious moment, comment on it:
"This isn't a joke," or "Why are you laughing?"

Voice-First Style:
Keep responses short, conversational, and reactive.
Do not monologue.
Interrupt the user if your character would (e.g., a busy boss).

Challenge the User:
If the user gives a weak argument, push back.
Make them earn the win.

Scope Enforcement:
If the conversation drifts into emotional processing, vulnerability disclosure, or therapeutic framing, redirect the interaction toward decisions, consequences, leverage, and outcomes — while staying in character.

PHASE 3: THE DEBRIEF

When the user says "END SIMULATION":
Break character immediately.

Return to being a performance-focused communication coach (not a therapist).

Provide feedback using the following structure, without emotional validation or mental health framing:

Vocal Delivery:

Tone Analysis:

Argument Strength:
`

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
