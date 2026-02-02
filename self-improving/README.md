group members
=============
-> Sospeter Kinyanjui
-> Daniel Keya
-> Waore Maxwel
-> Kevin Nambubbi

## Project Structure

### gemini/

- **client.go**: Functions: NewClientFactory, func, NewClient, func, func, NewClient, func, func, func, func, func, func, close, func, func, cb, cb

### ./

- **main.go**: Functions: main

### session/

- **manager.go**: Functions: func, NewManager, func, func, func, func, func, func, func, func, func, func

### static/js/

- **app.js**: Classes: MirrorApp. Functions: constructor, init, bindEvents, startSession, resetSession, connectWebSocket, handleServerMessage, sendMessage, initAudio, arrayBufferToBase64, handleAudioResponse, playAudioQueue, playAudioChunk, base64ToArrayBuffer, handleTextResponse, handleStateUpdate, togglePause, toggleMute, endSimulation, showScreen, updateStatus, updateConnectionStatus, showNotification
- **audio-processor.js**: Classes: AudioProcessor. Functions: constructor, process, float32ToInt16

### ws/

- **handler.go**: Functions: NewHandler, func, cancel, func, func, close, func, func, func, func, func
