Server Documentation: The Mirror Backend
[======================================]

This documentation covers the Go-based backend for The Mirror, a high-stakes conversation simulator. The server orchestrates real-time communication between a web frontend and the Gemini Live API via WebSockets.

Architecture Overview
<------------------->

The backend is built as a stateful WebSocket server. Each client connection maintains a dedicated, persistent connection to Google's Gemini Multimodal Live API.

    Gemini Package (/gemini): Manages the low-level WebSocket handshake with Google, audio encoding/decoding (PCM 16k), and system prompt injection.

    Session Package (/session): A state machine that tracks whether the user is in Setup, Simulation, or Debrief phases.

    WS Package (/ws): The glue layer that upgrades incoming HTTP requests and routes messages between the browser and Gemini.

    Main (main.go): Entry point that handles environment variables and starts the HTTPS server.

Core Components
===============

1. Gemini Live Integration (client.go)

The server uses the gemini-2.5-flash-native-audio-latest model to support real-time voice-to-voice interaction.

    System Prompt: Defines "The Mirror" persona, including rules for immersion and visual awareness (challenging the user's body language/tone).

    Audio Configuration: Uses audio/pcm;rate=16000 for low-latency streaming.

    Voice: Configured to use the Aoede prebuilt voice.

2. State Management (manager.go)

The Manager struct tracks the simulation lifecycle:

    Phases: PhaseSetup, PhaseSimulation, PhaseDebrief, and PhasePaused.

    Command Processing: Monitors text streams for keywords like "PAUSE" or "END SIMULATION" to trigger state transitions.

3. WebSocket Handler (handler.go)

Handles the BrowserMessage protocol:

    Audio Payload: Receives Base64 encoded PCM data from the frontend and forwards it to Gemini.

    Text Payload: Forwards user text or processes internal commands.

    Control Payload: Responds to manual UI triggers like "reset" or "resume".

Getting Started
===============

Prerequisites

    Go 1.20+

    A Gemini API Key (Vertex AI or AI Studio)

    SSL Certificates (mirror.pem and mirror-key.pem) for secure WebSocket (WSS) support.

Environment Variables

Create a .env file in the root directory:
Code snippet

GEMINI_API_KEY=your_api_key_here
PORT=8080

Installation & Running

    Install dependencies:
    Bash

    go get github.com/gorilla/websocket
    go get github.com/joho/godotenv

    Run the server:
    Bash

    go run main.go

API Protocol (Browser ↔ Server)
===============================

The frontend communicates with the server using JSON messages in the following format:
Message Type	Direction	Description
audio	Bidirectional	Base64 PCM audio data.
text	Bidirectional	Transcription or AI text response.
control	Frontend -> Backend	Actions: pause, resume, end, reset.
state	Backend -> Frontend	Syncs current phase and scenario details.
error	Backend -> Frontend	Reports connection or API failures.

Security Note
=============

The server is configured to use ListenAndServeTLS. Since browser microphone access requires a Secure Context, you must run this over HTTPS/WSS. For local development, ensure your self-signed certificates are trusted by your browser.
