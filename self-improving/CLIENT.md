#  MirrorApp – Frontend Application Overview

MirrorApp is the main frontend controller for the Mirror voice application. It handles microphone capture, audio streaming, WebSocket communication, audio playback, and UI state.

# Constructor

Initializes app state, audio variables, playback queue, and caches DOM elements. Calls init() to start setup.

# init()

Starts application setup by binding UI events.

# bindEvents()

Attaches click handlers for:

# Start → startSession()

Pause → togglePause()

# Mic → toggleMute()

# End → endSimulation()

Scenario card selection

# startSession()

Begins a new session:

Initializes audio

Connects WebSocket

Switches UI to simulation screen
Handles errors if microphone access fails.

# resetSession()

Resets the session:

Closes WebSocket and audio context

Stops microphone

Clears playback queue and flags

Returns UI to welcome screen

# connectWebSocket()

Connects to the server via WebSocket, handles:

onopen → connected

# onmessage → handleServerMessage()

onerror → error

onclose → disconnected

handleServerMessage(message)

Routes messages by type:

# audio → handleAudioResponse()

text → handleTextResponse()

state → handleStateUpdate()

notification → showNotification()

error → show error

sendMessage(type, payload)

Sends JSON message to server if connected.

# initAudio()

Initializes microphone capture and audio worklet:

Creates AudioContext

Loads worklet

Requests microphone

Streams audio to server via worklet

Handles base64 conversion

# arrayBufferToBase64(buffer)

Converts raw audio bytes to base64 for WebSocket transfer.

handleAudioResponse(payload)

Adds server audio to queue and starts playback. Updates speaking UI.

playAudioQueue()

Plays queued audio sequentially, respecting pause state. Updates UI while speaking.

# playAudioChunk(arrayBuffer)

Plays one audio chunk:

Converts Int16 PCM to Float32

Creates AudioBuffer

Plays and resolves when finished

# base64ToArrayBuffer(base64)

Decodes base64 server audio to binary buffer.

handleTextResponse(payload)

Appends AI text to transcript panel and scrolls transcript.

# handleStateUpdate(state)

Updates internal phase state (setup, simulation, paused, debrief) and UI accordingly.

# togglePause()

Pauses/resumes session:

Stops playback and audio sending

Updates UI and sends control message

# toggleMute()

Mutes/unmutes microphone:

Stops audio sending

Updates UI

# endSimulation()

Ends session completely:

Stops mic and audio

Closes audio context

Clears queue

Returns to welcome screen

Sends end message to server

# showScreen(screenName)

Switches UI between:

welcome

simulation

debrief
Hides inactive screens.

# updateStatus(text)

Updates small status message (Listening, Speaking, Paused).

# updateConnectionStatus(status)

Updates connection indicator in UI:

connected

error

disconnected

# showNotification(message)

Displays informational messages; currently logs to console.

Initialization
document.addEventListener('DOMContentLoaded', () => {
    window.mirrorApp = new MirrorApp();
});


Creates the MirrorApp instance when the page loads.