# AudioProcessor Worklet Documentation

# AudioProcessor is an AudioWorkletProcessor that captures real-time microphone audio, converts it to PCM, and sends it to the main thread for streaming.

# Constructor

Initializes a fixed buffer (bufferSize = 2048) to accumulate audio samples.

Sets bufferIndex to track how many samples are filled.

# process(inputs, outputs, parameters)

Called repeatedly by the audio engine.

Reads microphone input from inputs[0][0].

Stores samples in the buffer until it is full.

Once full:

Converts the buffer from Float32 to Int16 PCM (float32ToInt16).

Sends the PCM buffer to the main thread via this.port.postMessage.

Resets the buffer and bufferIndex.

Returns true to keep processing.

# float32ToInt16(float32Array)

Converts a Float32Array of audio samples (-1.0 to 1.0) to Int16Array.

Clamps values to [-1, 1] and scales to 16-bit PCM range.

Returns the Int16Array.

registerProcessor
# registerProcessor('audio-processor', AudioProcessor);


Registers the processor with the given name so it can be loaded into an AudioWorkletNode in the main thread.