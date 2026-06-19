/**
 * ElevenLabs TTS + Whisper STT providers.
 *
 * Architecture:
 *   - ElevenLabsTTS: Text-to-speech via ElevenLabs streaming API
 *   - WhisperSTT: Speech-to-text via OpenAI Whisper API
 *
 * These are SEPARATE providers — not a combined realtime provider.
 * Use when config.voice.provider === 'elevenlabs' + 'whisper'.
 *
 * ⚠️ These providers contain NO business logic.
 * They don't know about interviews, resumes, or job descriptions.
 * The caller supplies all behavioral configuration.
 */

import type {
  STTProvider,
  TTSProvider,
  TTSOptions,
  AudioChunk,
  TranscriptSegment,
} from '../types.js';

// ----- Configuration -----

/** ElevenLabs TTS configuration. */
export interface ElevenLabsTTSConfig {
  /** ElevenLabs API key */
  apiKey: string;
  /** Voice ID (from ElevenLabs voice library) */
  voiceId: string;
  /** Model ID (e.g., 'eleven_multilingual_v2', 'eleven_flash_v2_5') */
  modelId: string;
  /** Voice stability (0-1, default 0.5) */
  stability?: number;
  /** Similarity boost (0-1, default 0.75) */
  similarityBoost?: number;
  /** Output format (e.g., 'mp3_44100_128', 'pcm_24000') */
  outputFormat?: string;
}

/** Whisper STT configuration. */
export interface WhisperSTTConfig {
  /** OpenAI API key */
  apiKey: string;
  /** Model ID (e.g., 'whisper-1', 'gpt-4o-mini-transcribe') */
  modelId: string;
  /** Language hint (e.g., 'en') */
  language?: string;
  /** Prompt for better transcription of uncommon words */
  prompt?: string;
  /** Request timeout in ms (default 30000) */
  timeoutMs?: number;
  /** API endpoint override (for testing/self-hosted Whisper) */
  apiEndpoint?: string;
}

// ----- Errors -----

/** Error thrown when ElevenLabs API call fails. */
export class ElevenLabsError extends Error {
  constructor(
    message: string,
    public readonly statusCode?: number,
    cause?: unknown
  ) {
    super(message, { cause });
    this.name = 'ElevenLabsError';
  }
}

/** Error thrown when Whisper API call fails. */
export class WhisperError extends Error {
  constructor(
    message: string,
    public readonly statusCode?: number,
    cause?: unknown
  ) {
    super(message, { cause });
    this.name = 'WhisperError';
  }
}

// ----- SDK Adapter -----

/**
 * Minimal interface for ElevenLabs SDK functionality we use.
 * Defines the contract — adapter implements it safely.
 */
interface ElevenLabsSDK {
  textToSpeech: {
    stream: (
      voiceId: string,
      params: {
        text: string;
        model_id: string;
        output_format?: string;
        voice_settings?: {
          stability?: number;
          similarity_boost?: number;
        };
      }
    ) => Promise<ReadableStream<Uint8Array>>;
  };
}

/**
 * Safe adapter wrapping the dynamically imported ElevenLabs SDK.
 *
 * Avoids `as unknown as` double cast by wrapping the real client
 * in a thin class that delegates to the underlying methods.
 * If the SDK API surface changes, this adapter fails at compile time,
 * not at runtime.
 */
class ElevenLabsClientAdapter implements ElevenLabsSDK {
  readonly textToSpeech: ElevenLabsSDK['textToSpeech'];

  constructor(nativeClient: Record<string, unknown>) {
    // Extract and bind the textToSpeech methods from the native client
    const tts = nativeClient['textToSpeech'] as Record<string, unknown> | undefined;
    if (!tts || typeof tts['stream'] !== 'function') {
      throw new ElevenLabsError(
        'ElevenLabs SDK does not expose textToSpeech.stream() — API surface may have changed'
      );
    }

    this.textToSpeech = {
      stream: (voiceId, params) =>
        (tts['stream'] as (id: string, p: unknown) => Promise<ReadableStream<Uint8Array>>)(
          voiceId,
          params
        ),
    };
  }
}

// ----- ElevenLabs TTS -----

/**
 * ElevenLabs text-to-speech provider.
 *
 * Streams audio chunks from ElevenLabs API — never buffers entire response.
 * Uses the official @elevenlabs/elevenlabs-js SDK for connection management.
 *
 * Lifecycle: initialize() → synthesize() → destroy()
 */
export class ElevenLabsTTS implements TTSProvider {
  readonly name = 'elevenlabs';

  private readonly config: ElevenLabsTTSConfig;
  private client: ElevenLabsSDK | null = null;
  private initialized = false;
  private activeReaders = new Set<ReadableStreamDefaultReader<Uint8Array>>();

  constructor(config: ElevenLabsTTSConfig) {
    this.config = config;
  }

  /**
   * Initialize the ElevenLabs client.
   *
   * Creates the SDK client instance — no network call until synthesize().
   * Uses adapter pattern to avoid unsafe type casts.
   */
  async initialize(): Promise<void> {
    if (this.initialized) {
      return;
    }

    try {
      const { ElevenLabsClient } = await import('elevenlabs');
      const nativeClient = new ElevenLabsClient({
        apiKey: this.config.apiKey,
      });
      // Cast to unknown first — adapter validates the API surface at init time
      this.client = new ElevenLabsClientAdapter(nativeClient as unknown as Record<string, unknown>);
      this.initialized = true;
    } catch (error) {
      const err = error instanceof Error ? error : new Error(String(error));
      throw new ElevenLabsError(`Failed to initialize ElevenLabs: ${err.message}`, undefined, error);
    }
  }

  /**
   * Convert text to audio stream.
   *
   * Yields AudioChunk objects as they arrive from ElevenLabs.
   * Each chunk contains a structural copy of the audio — caller can safely queue it.
   *
   * Note: ElevenLabs streaming API does not support per-request speed control.
   * TTSOptions.speed is ignored — speed is determined by model and voice settings.
   * TTSOptions.language is also ignored — ElevenLabs uses multilingual models.
   *
   * @param text - Text to synthesize
   * @param options - Voice overrides (voice only — speed/language not supported)
   * @yields AudioChunk with PCM16 audio data
   * @throws ElevenLabsError if API call fails
   */
  async *synthesize(text: string, options?: TTSOptions): AsyncIterable<AudioChunk> {
    if (!this.initialized || !this.client) {
      throw new ElevenLabsError('ElevenLabs TTS not initialized — call initialize() first');
    }

    const voiceId = options?.voice ?? this.config.voiceId;
    const outputFormat = this.config.outputFormat ?? 'pcm_24000';

    try {
      // Use stream() for low-latency streaming
      const stream = await this.client.textToSpeech.stream(voiceId, {
        text,
        model_id: this.config.modelId,
        output_format: outputFormat,
        voice_settings: {
          stability: this.config.stability ?? 0.5,
          similarity_boost: this.config.similarityBoost ?? 0.75,
        },
      });

      // Track reader in Set for safe concurrent abort
      const reader = stream.getReader();
      this.activeReaders.add(reader);

      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          // Structural copy — drops reference to underlying network allocation
          // Prevents memory pinning if caller queues this AudioChunk
          const buffer = Buffer.from(value);

          yield {
            data: buffer,
            sampleRate: this.getSampleRate(outputFormat),
            channels: 1,
          };
        }
      } finally {
        this.activeReaders.delete(reader);
        reader.releaseLock();
      }
    } catch (error) {
      if (error instanceof ElevenLabsError) throw error;
      const err = error instanceof Error ? error : new Error(String(error));
      throw new ElevenLabsError(`ElevenLabs synthesis failed: ${err.message}`, undefined, error);
    }
  }

  /**
   * Clean up resources.
   *
   * Aborts ALL in-flight stream readers, then clears state.
   * Safe to call multiple times — idempotent.
   */
  async destroy(): Promise<void> {
    // Abort all active stream readers
    for (const reader of this.activeReaders) {
      try {
        await reader.cancel();
      } catch {
        // Ignore cancel errors during cleanup
      }
    }
    this.activeReaders.clear();

    this.client = null;
    this.initialized = false;
  }

  /** Extract sample rate from output format string. */
  private getSampleRate(outputFormat: string): number {
    const formatMap: Record<string, number> = {
      pcm_24000: 24000,
      pcm_16000: 16000,
      mp3_44100_128: 44100,
    };

    return formatMap[outputFormat] ?? 24000;
  }
}

// ----- Whisper STT -----

/**
 * OpenAI Whisper speech-to-text provider.
 *
 * Transcribes audio via OpenAI's Whisper API.
 * Uses raw HTTP for minimal dependencies — no OpenAI SDK required.
 *
 * Audio must be PCM16 16kHz mono or 24kHz mono.
 * Whisper accepts: flac, mp3, mp4, mpeg, mpga, m4a, ogg, wav, webm
 * We convert PCM16 Buffer to WAV before sending.
 *
 * Lifecycle: initialize() → transcribe() → destroy()
 */
export class WhisperSTT implements STTProvider {
  readonly name = 'whisper';

  private readonly config: WhisperSTTConfig;
  private readonly apiEndpoint: string;
  private readonly timeoutMs: number;
  private initialized = false;
  private activeControllers = new Set<AbortController>();

  constructor(config: WhisperSTTConfig) {
    this.config = config;
    this.apiEndpoint = config.apiEndpoint ?? 'https://api.openai.com/v1/audio/transcriptions';
    this.timeoutMs = config.timeoutMs ?? 30_000;
  }

  /**
   * Initialize the Whisper STT provider.
   *
   * No-op — Whisper is stateless. Kept for interface consistency.
   */
  async initialize(): Promise<void> {
    this.initialized = true;
  }

  /**
   * Transcribe audio chunks to text.
   *
   * Converts PCM16 to WAV, sends to Whisper API, returns transcript segment.
   * Returns null if transcription is empty.
   *
   * @param audio - Audio chunk to transcribe
   * @returns TranscriptSegment or null if empty
   * @throws WhisperError if API call fails or times out
   */
  async transcribe(audio: AudioChunk): Promise<TranscriptSegment | null> {
    if (!this.initialized) {
      throw new WhisperError('Whisper STT not initialized — call initialize() first');
    }

    // Validate input
    if (audio.channels < 1 || audio.channels > 2) {
      throw new WhisperError(`Unsupported channel count: ${audio.channels}`);
    }
    if (audio.sampleRate < 1 || audio.sampleRate > 192000) {
      throw new WhisperError(`Invalid sample rate: ${audio.sampleRate}`);
    }

    // Convert PCM16 to WAV for Whisper API
    const wavBuffer = this.pcm16ToWav(
      audio.data instanceof ArrayBuffer
        ? Buffer.from(audio.data)
        : audio.data,
      audio.sampleRate,
      audio.channels
    );

    // Create multipart form data
    const formData = new FormData();
    // Structural copy — ensures no SharedArrayBuffer backing
    const arrayBuffer = new ArrayBuffer(wavBuffer.byteLength);
    new Uint8Array(arrayBuffer).set(wavBuffer);
    const audioBlob = new Blob([arrayBuffer], { type: 'audio/wav' });
    formData.append('file', audioBlob, 'audio.wav');
    formData.append('model', this.config.modelId);
    formData.append('response_format', 'json');

    if (this.config.language) {
      formData.append('language', this.config.language);
    }

    if (this.config.prompt) {
      formData.append('prompt', this.config.prompt);
    }

    // Timeout-protected fetch with abort tracking
    const controller = new AbortController();
    this.activeControllers.add(controller);
    const timeout = setTimeout(() => controller.abort(), this.timeoutMs);

    try {
      const response = await fetch(this.apiEndpoint, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${this.config.apiKey}`,
        },
        body: formData,
        signal: controller.signal,
      });

      if (!response.ok) {
        const errorBody = await response.text().catch(() => 'unknown');
        throw new WhisperError(
          `Whisper API error: ${response.status} ${response.statusText} — ${errorBody}`,
          response.status
        );
      }

      const result = (await response.json()) as { text: string };

      if (!result.text || result.text.trim().length === 0) {
        return null;
      }

      return {
        speaker: 'user',
        text: result.text.trim(),
        timestamp: new Date().toISOString(),
      };
    } catch (error) {
      if (error instanceof WhisperError) throw error;

      // Handle AbortError (timeout or destroy-time abort)
      if (error instanceof DOMException && error.name === 'AbortError') {
        throw new WhisperError(`Whisper API request aborted (timeout: ${this.timeoutMs}ms)`);
      }

      const err = error instanceof Error ? error : new Error(String(error));
      throw new WhisperError(`Whisper transcription failed: ${err.message}`, undefined, error);
    } finally {
      clearTimeout(timeout);
      this.activeControllers.delete(controller);
    }
  }

  /**
   * Clean up resources.
   *
   * Aborts ALL in-flight transcription requests, then clears state.
   * Safe to call multiple times — idempotent.
   */
  async destroy(): Promise<void> {
    // Abort all active fetch requests
    for (const controller of this.activeControllers) {
      controller.abort();
    }
    this.activeControllers.clear();

    this.initialized = false;
  }

  /**
   * Convert PCM16 audio to WAV format.
   *
   * Whisper API expects WAV/MP3/FLAC — not raw PCM.
   * This creates a minimal WAV header for the PCM data.
   *
   * @param pcmData - Raw PCM16 audio data
   * @param sampleRate - Sample rate in Hz (e.g., 16000, 24000)
   * @param channels - Number of channels (1 = mono, 2 = stereo)
   * @returns WAV-formatted Buffer
   * @throws WhisperError if input is invalid
   */
  private pcm16ToWav(pcmData: Buffer, sampleRate: number, channels: number): Buffer {
    // Input validation
    if (channels < 1 || channels > 2) {
      throw new WhisperError(`Unsupported channel count for WAV: ${channels}`);
    }
    if (sampleRate < 1 || sampleRate > 192000) {
      throw new WhisperError(`Invalid sample rate for WAV: ${sampleRate}`);
    }
    if (pcmData.length > 0xFFFFFFFF) {
      throw new WhisperError('PCM data too large for WAV format (>4GB)');
    }

    const bitsPerSample = 16;
    const bytesPerSample = bitsPerSample / 8;
    const blockAlign = channels * bytesPerSample;
    const byteRate = sampleRate * blockAlign;
    const dataSize = pcmData.length;
    const headerSize = 44;

    const header = Buffer.alloc(headerSize);

    // RIFF header
    header.write('RIFF', 0);
    header.writeUInt32LE(36 + dataSize, 4); // file size - 8
    header.write('WAVE', 8);

    // fmt sub-chunk
    header.write('fmt ', 12);
    header.writeUInt32LE(16, 16); // sub-chunk size
    header.writeUInt16LE(1, 20); // PCM format
    header.writeUInt16LE(channels, 22);
    header.writeUInt32LE(sampleRate, 24);
    header.writeUInt32LE(byteRate, 28);
    header.writeUInt16LE(blockAlign, 32);
    header.writeUInt16LE(bitsPerSample, 34);

    // data sub-chunk
    header.write('data', 36);
    header.writeUInt32LE(dataSize, 40);

    return Buffer.concat([header, pcmData]);
  }
}

// ----- Factory Functions -----

/**
 * Create an ElevenLabs TTS provider.
 *
 * @param config - ElevenLabs configuration (API key, voice, model)
 * @returns Configured TTSProvider instance
 */
export function createElevenLabsTTS(config: ElevenLabsTTSConfig): TTSProvider {
  return new ElevenLabsTTS(config);
}

/**
 * Create a Whisper STT provider.
 *
 * @param config - Whisper configuration (API key, model, language)
 * @returns Configured STTProvider instance
 */
export function createWhisperSTT(config: WhisperSTTConfig): STTProvider {
  return new WhisperSTT(config);
}
