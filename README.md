# The Mirror - Conversation Simulator

Practice difficult conversations with AI-powered roleplay coaching using the Gemini Live API.

## Quick Start

### Prerequisites
- Go 1.21+
- A Gemini API key with Live API access

### Run Locally

```bash
# Clone and enter directory
cd mirror

# Set your API key
export GEMINI_API_KEY="your-api-key-here"

# Run the server
go run main.go
```

Open http://localhost:8080 in your browser.

## Usage

1. **Start Session** - Click the button and allow microphone access
2. **Setup** - Tell the AI who to roleplay (e.g., "You're my manager David, I want to ask for a raise")
3. **Choose Difficulty** - Easy, Realistic, or Hostile
4. **Practice** - The AI becomes that character immediately
5. **Control Commands**:
   - Say "PAUSE" to pause the simulation
   - Say "END SIMULATION" to get feedback
6. **Debrief** - Receive coaching feedback on body language, tone, and arguments

## Deployment (VPS)

### Using Docker

```bash
docker build -t mirror .
docker run -p 8080:8080 -e GEMINI_API_KEY="your-key" mirror
```

### Manual

```bash
go build -o mirror .
GEMINI_API_KEY="your-key" ./mirror
```

## Project Structure

```
mirror/
├── main.go              # Entry point
├── gemini/
│   └── client.go        # Gemini Live API WebSocket client
├── session/
│   └── manager.go       # Phase state machine
├── ws/
│   └── handler.go       # Browser WebSocket handler
└── static/
    ├── index.html       # Main UI
    ├── css/style.css    # Styling
    └── js/
        ├── app.js           # Main application
        └── audio-processor.js  # Audio capture worklet
```
