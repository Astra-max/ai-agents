/**
 * The Mirror - Main Application
 * Handles WebSocket communication, audio streaming, and UI state
 */

class MirrorApp {
    constructor() {
        // State
        this.ws = null;
        this.audioContext = null;
        this.mediaStream = null;
        this.workletNode = null;
        this.isConnected = false;
        this.isMuted = false;
        this.isPaused = false;
        this.currentPhase = 'setup';

        // Audio playback
        this.audioQueue = [];
        this.isPlaying = false;

        // DOM Elements
        this.elements = {
            welcomeScreen: document.getElementById('welcomeScreen'),
            simulationScreen: document.getElementById('simulationScreen'),
            debriefScreen: document.getElementById('debriefScreen'),
            startBtn: document.getElementById('startBtn'),
            pauseBtn: document.getElementById('pauseBtn'),
            micBtn: document.getElementById('micBtn'),
            endBtn: document.getElementById('endBtn'),
            newSessionBtn: document.getElementById('newSessionBtn'),
            sessionIndicator: document.getElementById('sessionIndicator'),
            connectionStatus: document.getElementById('connectionStatus'),
            aiAvatar: document.getElementById('aiAvatar'),
            avatarEmoji: document.getElementById('avatarEmoji'),
            characterName: document.getElementById('characterName'),
            audioVisualizer: document.getElementById('audioVisualizer'),
            statusIndicator: document.getElementById('statusIndicator'),
            transcript: document.getElementById('transcript'),
            debriefContent: document.getElementById('debriefContent'),
            scenarioCards: document.querySelectorAll('.scenario-card')
        };

        this.init();
    }

    init() {
        this.bindEvents();
    }

    bindEvents() {
        // Start button
        this.elements.startBtn.addEventListener('click', () => this.startSession());

        // Control buttons
        this.elements.pauseBtn.addEventListener('click', () => this.togglePause());
        this.elements.micBtn.addEventListener('click', () => this.toggleMute());
        this.elements.endBtn.addEventListener('click', () => this.endSimulation());
        this.elements.newSessionBtn.addEventListener('click', () => this.resetSession());

        // Scenario cards
        this.elements.scenarioCards.forEach(card => {
            card.addEventListener('click', () => {
                this.elements.scenarioCards.forEach(c => c.classList.remove('selected'));
                card.classList.add('selected');
            });
        });
    }

    // ==================
    // Session Management
    // ==================

    async startSession() {
        try {
            // Request microphone access
            await this.initAudio();

            // Connect to WebSocket
            await this.connectWebSocket();

            // Switch to simulation screen
            this.showScreen('simulation');
            this.updateStatus('Connecting to your coach...');

        } catch (error) {
            console.error('Failed to start session:', error);
            alert('Failed to start session. Please ensure microphone access is granted.');
        }
    }

    resetSession() {
        // Close connections
        if (this.ws) {
            this.ws.close();
        }
        if (this.mediaStream) {
            this.mediaStream.getTracks().forEach(track => track.stop());
        }
        if (this.audioContext) {
            this.audioContext.close();
        }

        // Reset state
        this.isConnected = false;
        this.isMuted = false;
        this.isPaused = false;
        this.currentPhase = 'setup';
        this.audioQueue = [];

        // Show welcome screen
        this.showScreen('welcome');
        this.updateConnectionStatus('disconnected');
    }

    // ==================
    // WebSocket
    // ==================

    async connectWebSocket() {
        return new Promise((resolve, reject) => {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws`;

            this.ws = new WebSocket(wsUrl);

            this.ws.onopen = () => {
                console.log('WebSocket connected');
                this.isConnected = true;
                this.updateConnectionStatus('connected');
                resolve();
            };

            this.ws.onmessage = (event) => {
                const message = JSON.parse(event.data);
                this.handleServerMessage(message);
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.updateConnectionStatus('error');
                reject(error);
            };

            this.ws.onclose = () => {
                console.log('WebSocket closed');
                this.isConnected = false;
                this.updateConnectionStatus('disconnected');
            };
        });
    }

    handleServerMessage(message) {
        console.log('Server message:', message.type);

        switch (message.type) {
            case 'audio':
                this.handleAudioResponse(message.payload);
                break;

            case 'text':
                this.handleTextResponse(message.payload);
                break;

            case 'state':
                this.handleStateUpdate(message.payload);
                break;

            case 'notification':
                this.showNotification(message.payload.message);
                break;

            case 'error':
                console.error('Server error:', message.error);
                this.showNotification('Error: ' + message.error);
                break;
        }
    }

    sendMessage(type, payload) {
        if (this.ws && this.isConnected) {
            this.ws.send(JSON.stringify({ type, payload }));
        }
    }

    // ==================
    // Audio Capture
    // ==================

    async initAudio() {
        // Get microphone access
        this.mediaStream = await navigator.mediaDevices.getUserMedia({
            audio: {
                sampleRate: 16000,
                channelCount: 1,
                echoCancellation: true,
                noiseSuppression: true
            }
        });

        // Create audio context at 16kHz
        this.audioContext = new AudioContext({ sampleRate: 16000 });

        // Load audio worklet
        await this.audioContext.audioWorklet.addModule('js/audio-processor.js');

        // Create source from microphone
        const source = this.audioContext.createMediaStreamSource(this.mediaStream);

        // Create worklet node
        this.workletNode = new AudioWorkletNode(this.audioContext, 'audio-processor');

        // Handle audio data from worklet
        this.workletNode.port.onmessage = (event) => {
            if (event.data.type === 'audio' && !this.isMuted && !this.isPaused) {
                // Convert ArrayBuffer to base64
                const base64 = this.arrayBufferToBase64(event.data.data);
                this.sendMessage('audio', { data: base64 });
            }
        };

        // Connect: mic -> worklet
        source.connect(this.workletNode);
        // Don't connect to destination (we don't want to hear ourselves)
    }

    arrayBufferToBase64(buffer) {
        const bytes = new Uint8Array(buffer);
        let binary = '';
        for (let i = 0; i < bytes.byteLength; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        return btoa(binary);
    }

    // ==================
    // Audio Playback
    // ==================

    async handleAudioResponse(payload) {
        // Decode base64 audio
        const audioData = this.base64ToArrayBuffer(payload.data);

        // Add to queue
        this.audioQueue.push(audioData);

        // Start playback if not already playing
        if (!this.isPlaying) {
            this.playAudioQueue();
        }

        // Show speaking animation
        this.elements.aiAvatar.classList.add('speaking');
        this.elements.audioVisualizer.classList.add('active');
        this.updateStatus('Speaking...');
    }

    async playAudioQueue() {
        this.isPlaying = true;

        while (this.audioQueue.length > 0) {
            const audioData = this.audioQueue.shift();
            await this.playAudioChunk(audioData);
        }

        this.isPlaying = false;
        this.elements.aiAvatar.classList.remove('speaking');
        this.elements.audioVisualizer.classList.remove('active');
        this.updateStatus('Listening...');
    }

    async playAudioChunk(arrayBuffer) {
        return new Promise((resolve) => {
            // Create audio context for playback at 24kHz (Gemini output rate)
            const playbackContext = new AudioContext({ sampleRate: 24000 });

            // Convert Int16 PCM to Float32
            const int16Array = new Int16Array(arrayBuffer);
            const float32Array = new Float32Array(int16Array.length);

            for (let i = 0; i < int16Array.length; i++) {
                float32Array[i] = int16Array[i] / 32768.0;
            }

            // Create AudioBuffer
            const audioBuffer = playbackContext.createBuffer(1, float32Array.length, 24000);
            audioBuffer.getChannelData(0).set(float32Array);

            // Play
            const source = playbackContext.createBufferSource();
            source.buffer = audioBuffer;
            source.connect(playbackContext.destination);
            source.onended = () => {
                playbackContext.close();
                resolve();
            };
            source.start();
        });
    }

    base64ToArrayBuffer(base64) {
        const binary = atob(base64);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
            bytes[i] = binary.charCodeAt(i);
        }
        return bytes.buffer;
    }

    // ==================
    // Response Handlers
    // ==================

    handleTextResponse(payload) {
        console.log('AI text:', payload.text);

        // Update transcript
        const transcriptContent = this.elements.transcript.querySelector('.transcript-content');
        transcriptContent.innerHTML += `<p><strong>Coach:</strong> ${payload.text}</p>`;
        this.elements.transcript.classList.add('visible');
        this.elements.transcript.scrollTop = this.elements.transcript.scrollHeight;
    }

    handleStateUpdate(state) {
        console.log('State update:', state);
        this.currentPhase = state.phase;

        // Update phase indicator
        const phaseText = this.elements.sessionIndicator.querySelector('.phase-text');
        const phaseDot = this.elements.sessionIndicator.querySelector('.phase-dot');

        phaseText.textContent = state.phase.charAt(0).toUpperCase() + state.phase.slice(1);
        phaseDot.classList.remove('active', 'recording');

        if (state.phase === 'simulation') {
            phaseDot.classList.add('recording');
        } else if (state.phase !== 'paused') {
            phaseDot.classList.add('active');
        }

        // Handle phase transitions
        if (state.phase === 'debrief') {
            this.showScreen('debrief');
        }
    }

    // ==================
    // Controls
    // ==================

    togglePause() {
        this.isPaused = !this.isPaused;

        const icon = this.elements.pauseBtn.querySelector('.control-icon');
        icon.textContent = this.isPaused ? '▶️' : '⏸️';

        this.sendMessage('control', { action: this.isPaused ? 'pause' : 'resume' });
        this.updateStatus(this.isPaused ? 'Paused' : 'Listening...');
    }

    toggleMute() {
        this.isMuted = !this.isMuted;

        this.elements.micBtn.classList.toggle('muted', this.isMuted);
        const icon = this.elements.micBtn.querySelector('.control-icon');
        icon.textContent = this.isMuted ? '🔇' : '🎙️';

        this.updateStatus(this.isMuted ? 'Muted' : 'Listening...');
    }

    endSimulation() {
        this.sendMessage('control', { action: 'end' });
    }

    // ==================
    // UI Updates
    // ==================

    showScreen(screenName) {
        // Hide all screens
        this.elements.welcomeScreen.classList.remove('active');
        this.elements.simulationScreen.classList.remove('active');
        this.elements.debriefScreen.classList.remove('active');

        // Show requested screen
        switch (screenName) {
            case 'welcome':
                this.elements.welcomeScreen.classList.add('active');
                break;
            case 'simulation':
                this.elements.simulationScreen.classList.add('active');
                break;
            case 'debrief':
                this.elements.debriefScreen.classList.add('active');
                break;
        }
    }

    updateStatus(text) {
        const statusText = this.elements.statusIndicator.querySelector('.status-text');
        statusText.textContent = text;
    }

    updateConnectionStatus(status) {
        const statusEl = this.elements.connectionStatus;
        const textEl = statusEl.querySelector('.connection-text');

        statusEl.classList.remove('connected', 'error', 'hidden');

        switch (status) {
            case 'connected':
                statusEl.classList.add('connected');
                textEl.textContent = 'Connected';
                // Hide after 2 seconds
                setTimeout(() => statusEl.classList.add('hidden'), 2000);
                break;
            case 'error':
                statusEl.classList.add('error');
                textEl.textContent = 'Connection error';
                break;
            case 'disconnected':
                textEl.textContent = 'Disconnected';
                break;
            default:
                textEl.textContent = 'Connecting...';
        }
    }

    showNotification(message) {
        console.log('Notification:', message);
        // Could implement a toast notification here
    }
}

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.mirrorApp = new MirrorApp();
});
