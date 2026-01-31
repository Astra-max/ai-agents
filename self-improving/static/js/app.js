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
        this.elements.startBtn.addEventListener('click', () => this.startSession());
        this.elements.pauseBtn.addEventListener('click', () => this.togglePause());
        this.elements.micBtn.addEventListener('click', () => this.toggleMute());
        this.elements.endBtn.addEventListener('click', () => this.endSimulation());
        this.elements.newSessionBtn.addEventListener('click', () => this.resetSession());

        this.elements.scenarioCards.forEach(card => {
            card.addEventListener('click', () => {
                this.elements.scenarioCards.forEach(c => c.classList.remove('selected'));
                card.classList.add('selected');
            });
        });
    }

    async startSession() {
        try {
            await this.initAudio();
            await this.connectWebSocket();
            this.showScreen('simulation');
            this.updateStatus('Connecting to your coach...');
        } catch (error) {
            console.error('Failed to start session:', error);
            alert('Failed to start session. Please ensure microphone access is granted.');
        }
    }

    resetSession() {
        if (this.ws) this.ws.close();
        if (this.mediaStream) this.mediaStream.getTracks().forEach(track => track.stop());
        if (this.audioContext) this.audioContext.close();

        this.isConnected = false;
        this.isMuted = false;
        this.isPaused = false;
        this.currentPhase = 'setup';
        this.audioQueue = [];

        this.showScreen('welcome');
        this.updateConnectionStatus('disconnected');
    }

    async connectWebSocket() {
        return new Promise((resolve, reject) => {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws`;

            this.ws = new WebSocket(wsUrl);

            this.ws.onopen = () => {
                this.isConnected = true;
                this.updateConnectionStatus('connected');
                resolve();
            };

            this.ws.onmessage = (event) => {
                const message = JSON.parse(event.data);
                this.handleServerMessage(message);
            };

            this.ws.onerror = (error) => {
                this.updateConnectionStatus('error');
                reject(error);
            };

            this.ws.onclose = () => {
                this.isConnected = false;
                this.updateConnectionStatus('disconnected');
            };
        });
    }

    handleServerMessage(message) {
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
    // Audio Capture (FIXED)
    // ==================

    async initAudio() {
        // Create audio context FIRST (user gesture)
        this.audioContext = new AudioContext({ sampleRate: 16000 });

        if (this.audioContext.state === 'suspended') {
            await this.audioContext.resume();
        }

        // Load audio worklet
        await this.audioContext.audioWorklet.addModule('js/audio-processor.js');

        // Now request microphone access
        this.mediaStream = await navigator.mediaDevices.getUserMedia({
            audio: {
                sampleRate: 16000,
                channelCount: 1,
                echoCancellation: true,
                noiseSuppression: true
            }
        });

        const source = this.audioContext.createMediaStreamSource(this.mediaStream);
        this.workletNode = new AudioWorkletNode(this.audioContext, 'audio-processor');

        this.workletNode.port.onmessage = (event) => {
            if (event.data.type === 'audio' && !this.isMuted && !this.isPaused) {
                const base64 = this.arrayBufferToBase64(event.data.data);
                this.sendMessage('audio', { data: base64 });
            }
        };

        source.connect(this.workletNode);
        this.workletNode.connect(this.audioContext.destination);
        this.audioContext.destination.channelCount = 1;
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
        const audioData = this.base64ToArrayBuffer(payload.data);
        this.audioQueue.push(audioData);

        if (!this.isPlaying) {
            this.playAudioQueue();
        }

        this.elements.aiAvatar.classList.add('speaking');
        this.elements.audioVisualizer.classList.add('active');
        this.updateStatus('Speaking...');
    }

    async playAudioQueue() {
        this.isPlaying = true;

        while (this.audioQueue.length > 0) {
            // New logic: If paused, wait here instead of playing the next chunk
            if (this.isPaused) {
                await new Promise(r => setTimeout(r, 100)); 
                continue;
            }

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
            const playbackContext = new AudioContext({ sampleRate: 24000 });

            const int16Array = new Int16Array(arrayBuffer);
            const float32Array = new Float32Array(int16Array.length);

            for (let i = 0; i < int16Array.length; i++) {
                float32Array[i] = int16Array[i] / 32768.0;
            }

            const audioBuffer = playbackContext.createBuffer(1, float32Array.length, 24000);
            audioBuffer.getChannelData(0).set(float32Array);

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

    handleTextResponse(payload) {
        const transcriptContent = this.elements.transcript.querySelector('.transcript-content');
        transcriptContent.innerHTML += `<p><strong>Coach:</strong> ${payload.text}</p>`;
        this.elements.transcript.classList.add('visible');
        this.elements.transcript.scrollTop = this.elements.transcript.scrollHeight;
    }

    handleStateUpdate(state) {
        this.currentPhase = state.phase;
    }

    // Replace your existing togglePause with this:
    togglePause() {
        this.isPaused = !this.isPaused;
        
        // Visual Feedback
        if (this.isPaused) {
            this.elements.pauseBtn.textContent = 'Resume';
            this.elements.pauseBtn.style.backgroundColor = '#ff9800'; // Orange for pause
            this.updateStatus('Simulation Paused');
        } else {
            this.elements.pauseBtn.textContent = 'Pause';
            this.elements.pauseBtn.style.backgroundColor = ''; // Reset to default
            this.updateStatus('Listening...');
        }

        // Send control message to Go server
        this.sendMessage('control', { action: this.isPaused ? 'pause' : 'resume' });
    }

    // Replace your existing toggleMute with this:
    toggleMute() {
        this.isMuted = !this.isMuted;
        
        // Visual Feedback
        if (this.isMuted) {
            this.elements.micBtn.textContent = 'Unmute Mic';
            this.elements.micBtn.style.backgroundColor = '#f44336'; // Red for muted
            this.updateStatus('Microphone Muted');
        } else {
            this.elements.micBtn.textContent = 'Mute Mic';
            this.elements.micBtn.style.backgroundColor = ''; // Reset to default
            this.updateStatus('Listening...');
        }
    }

    // Replace your existing endSimulation with this:
    endSimulation() {
    // 1. Tell the server to stop
    this.sendMessage('control', { action: 'end' });
    
    // 2. Stop the local microphone and audio context
    if (this.mediaStream) {
        this.mediaStream.getTracks().forEach(track => track.stop());
    }
    if (this.audioContext) {
        this.audioContext.close();
    }

    // 3. Switch the UI back to the welcome screen
    this.showScreen('welcome'); 
    
    // 4. Reset internal states so the next session starts fresh
    this.isConnected = false;
    this.audioQueue = [];
    this.updateStatus('Session ended.');
    this.updateConnectionStatus('disconnected');
}

    showScreen(screenName) {
        this.elements.welcomeScreen.classList.remove('active');
        this.elements.simulationScreen.classList.remove('active');
        this.elements.debriefScreen.classList.remove('active');

        if (screenName === 'welcome') this.elements.welcomeScreen.classList.add('active');
        if (screenName === 'simulation') this.elements.simulationScreen.classList.add('active');
        if (screenName === 'debrief') this.elements.debriefScreen.classList.add('active');
    }

    updateStatus(text) {
        this.elements.statusIndicator.querySelector('.status-text').textContent = text;
    }

    updateConnectionStatus(status) {
        const statusEl = this.elements.connectionStatus;
        const textEl = statusEl.querySelector('.connection-text');

        statusEl.classList.remove('connected', 'error', 'hidden');

        if (status === 'connected') {
            statusEl.classList.add('connected');
            textEl.textContent = 'Connected';
            setTimeout(() => statusEl.classList.add('hidden'), 2000);
        } else if (status === 'error') {
            statusEl.classList.add('error');
            textEl.textContent = 'Connection error';
        } else {
            textEl.textContent = 'Disconnected';
        }
    }

    showNotification(message) {
        console.log('Notification:', message);
    }
}

document.addEventListener('DOMContentLoaded', () => {
    window.mirrorApp = new MirrorApp();
});
