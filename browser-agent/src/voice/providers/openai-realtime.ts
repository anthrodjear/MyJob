/**
 * OpenAI Realtime API provider — combined STT + TTS via WebSocket.
 *
 * Uses @openai/realtime-api-beta reference client for connection management.
 * Audio format: PCM16 24000Hz mono (OpenAI Realtime requirement).
 *
 * Lifecycle: connect() → sendAudio()/speak()/cancel() → disconnect() → destroy()
 *
 * ⚠️ This provider handles BOTH STT and TTS in one WebSocket connection.
 * Use when config.voice.provider === 'openai_realtime'.
 * For separate STT/TTS providers, use whisper + elevenlabs instead.
 *
 * ⚠️ This provider contains NO business logic.
 * It doesn't know if it's powering an interview agent, customer support bot,
 * or sales assistant. The caller supplies all behavioral configuration.
 */

import { EventEmitter } from 'events';
import type {
  RealtimeProvider,
  RealtimeProviderEventMap,
  AudioChunk,
  TranscriptSegment,
} from '../types.js';

/**
 * Configuration for OpenAI Realtime provider.
 * All tuning values are injected by the caller — nothing is hardcoded here.
 */
export interface OpenAIRealtimeConfig {
  /** OpenAI API key */
  apiKey: string;
  /** Model name (e.g., 'gpt-4o-realtime-preview') */
  model: string;
  /** TTS voice (e.g., 'alloy', 'echo', 'fable', 'onyx', 'nova', 'shimmer') */
  voice: string;
  /** STT transcription model (e.g., 'whisper-1') */
  transcriptionModel: string;
  /** Instructions for the realtime model — caller provides, not provider */
  instructions: string;
  /** Voice Activity Detection settings */
  vad: {
    threshold: number;
    prefixPaddingMs: number;
    silenceDurationMs: number;
  };
}

/**
 * Minimal interface for the RealtimeClient from @openai/realtime-api-beta.
 * Explicit structural types — no `unknown` parameters.
 */
interface RealtimeClientLike {
  connect(): Promise<void>;
  close(): Promise<void>;
  send(event: Record<string, unknown>): void;
  sendAudio(audio: ArrayBuffer): void;
  updateSession(config: Record<string, unknown>): Promise<void>;
  on(event: string, handler: (...args: unknown[]) => void): void;
  removeAllListeners(): void;
}

/**
 * Generate a unique ID for tracking pending operations.
 * Uses crypto.randomUUID when available, falls back to timestamp + random.
 */
function generateClientId(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return `msg_${Date.now()}_${Math.random().toString(36).slice(2, 9)}`;
}

/**
 * OpenAI Realtime API provider.
 *
 * Wraps @openai/realtime-api-beta to implement the RealtimeProvider interface.
 * Pure transport — no business logic, no fallback prompts, no interview knowledge.
 *
 * Event flow:
 *   Audio in → server processes → transcript events + audio out
 *   Text in (speak()) → server synthesizes → audio out
 */
export class OpenAIRealtimeProvider implements RealtimeProvider {
  readonly name = 'openai-realtime';

  private readonly config: OpenAIRealtimeConfig;
  private readonly emitter = new EventEmitter();
  private client: RealtimeClientLike | null = null;
  private connected = false;
  private destroyed = false;
  private connecting: Promise<void> | null = null;

  /** Track pending speak operations by client ID */
  private pendingSpokes = new Map<string, { text: string; timestamp: number }>();

  constructor(config: OpenAIRealtimeConfig) {
    this.config = config;

    // Log unhandled errors — never silently swallow
    this.emitter.on('error', (err: unknown) => {
      const error = err instanceof Error ? err : new Error(String(err));
      console.error('[openai-realtime] unhandled provider error:', error.message);
    });
  }

  /**
   * Connect to OpenAI Realtime API via WebSocket.
   *
   * Creates a RealtimeClient, configures session, and wires event handlers.
   * Must be called before sendAudio() or speak().
   *
   * Concurrency-safe: multiple concurrent calls will share the same connection promise.
   * On failure, this.connecting resets so subsequent calls can retry.
   *
   * @throws Error if already destroyed, or connection fails
   */
  async connect(): Promise<void> {
    if (this.destroyed) {
      throw new Error('Provider has been destroyed');
    }

    if (this.connected) {
      return;
    }

    // Share connection promise if concurrent calls happen
    if (this.connecting) {
      return this.connecting;
    }

    this.connecting = (async () => {
      try {
        // Dynamic import — @openai/realtime-api-beta is ESM-only
        const { RealtimeClient } = await import('@openai/realtime-api-beta');

        const rawClient = new RealtimeClient({
          apiKey: this.config.apiKey,
          model: this.config.model,
        });

        this.client = rawClient as RealtimeClientLike;
        this.wireEvents();

        // Connect WebSocket
        await this.client.connect();
        this.connected = true;

        // Configure session
        await this.configureSession();

        this.emitter.emit('connected');
      } catch (error) {
        this.connected = false;
        this.client = null;
        const err = error instanceof Error ? error : new Error(String(error));
        this.emitter.emit('error', err);
        throw err;
      } finally {
        // Always reset connecting so subsequent calls can retry
        this.connecting = null;
      }
    })();

    return this.connecting;
  }

  /**
   * Disconnect from OpenAI Realtime API.
   *
   * Closes WebSocket and cleans up event handlers.
   * Safe to call multiple times.
   */
  async disconnect(): Promise<void> {
    if (!this.connected || !this.client) {
      return;
    }

    try {
      await this.client.close();
      this.client.removeAllListeners();
    } catch (error) {
      // Log cleanup errors for debugging
      const err = error instanceof Error ? error : new Error(String(error));
      console.debug('[openai-realtime] disconnect cleanup error:', err.message);
    } finally {
      this.connected = false;
      this.client = null;
      this.pendingSpokes.clear();
      this.emitter.emit('disconnected');
    }
  }

  /**
   * Send audio chunk to OpenAI for processing.
   *
   * Audio must be PCM16 24000Hz mono. If input is different format,
   * conversion should happen at the session layer before calling this.
   *
   * Uses Buffer.slice() to avoid unnecessary allocation — shares underlying memory.
   *
   * @throws Error if not connected or send fails
   */
  async sendAudio(audio: AudioChunk): Promise<void> {
    if (!this.connected || !this.client) {
      throw new Error('Not connected to OpenAI Realtime API');
    }

    // Fast path: already an ArrayBuffer — pass directly
    if (audio.data instanceof ArrayBuffer) {
      this.client.sendAudio(audio.data);
      return;
    }

    // Buffer path: try to get a view into the same memory without allocation
    const rawSlice = audio.data.buffer.slice(
      audio.data.byteOffset,
      audio.data.byteOffset + audio.data.byteLength
    );

    // Ensure we have a proper ArrayBuffer (not SharedArrayBuffer)
    // If the source is SharedArrayBuffer, we must copy to a new ArrayBuffer
    if (rawSlice instanceof ArrayBuffer) {
      this.client.sendAudio(rawSlice);
    } else {
      const arrayBuffer = new ArrayBuffer(audio.data.byteLength);
      new Uint8Array(arrayBuffer).set(audio.data);
      this.client.sendAudio(arrayBuffer);
    }
  }

  /**
   * Send text for the provider to speak via TTS.
   *
   * Creates a server-side response with text that gets synthesized to audio.
   * Use for: agent responses, system prompts, interruptions.
   *
   * Tracks pending operations by client ID for status monitoring.
   *
   * @throws Error if not connected or send fails
   */
  async speak(text: string): Promise<void> {
    if (!this.connected || !this.client) {
      throw new Error('Not connected to OpenAI Realtime API');
    }

    const clientId = generateClientId();

    // Track this speak operation
    this.pendingSpokes.set(clientId, {
      text,
      timestamp: Date.now(),
    });

    // Send text as a user message — server will respond with audio
    this.client.send({
      type: 'conversation.item.create',
      item: {
        type: 'message',
        role: 'user',
        content: [{ type: 'input_text', text }],
        id: clientId,
      },
    });

    // Trigger response generation — link to the same client ID
    this.client.send({
      type: 'response.create',
      response: { conversation_item_id: clientId },
    });
  }

  /**
   * Cancel ongoing audio output.
   *
   * Sends a cancellation event to stop the server from speaking.
   * Use when: candidate interrupts, agent needs to revise response.
   *
   * Logs errors for debugging — barge-in failures are critical in voice apps.
   */
  async cancel(): Promise<void> {
    if (!this.connected || !this.client) {
      return;
    }

    try {
      this.client.send({ type: 'response.cancel' });
    } catch (error) {
      // Log but don't throw — barge-in failures cause agent to speak over user
      const err = error instanceof Error ? error : new Error(String(error));
      console.warn('[openai-realtime] cancel failed:', err.message);
    }
  }

  /** Subscribe to realtime provider events with type-safe callback. */
  on<K extends keyof RealtimeProviderEventMap>(
    event: K,
    handler: RealtimeProviderEventMap[K]
  ): void {
    this.emitter.on(event, handler);
  }

  /** Remove event handler. */
  off<K extends keyof RealtimeProviderEventMap>(
    event: K,
    handler: RealtimeProviderEventMap[K]
  ): void {
    this.emitter.off(event, handler);
  }

  /**
   * Clean up all resources.
   *
   * Disconnects from WebSocket and removes all event listeners.
   * Safe to call multiple times — idempotent via destroyed flag.
   */
  async destroy(): Promise<void> {
    if (this.destroyed) {
      return;
    }

    this.destroyed = true;
    await this.disconnect();
    this.emitter.removeAllListeners();
  }

  // ----- Private methods -----

  /**
   * Wire event handlers from RealtimeClient to our EventEmitter.
   *
   * Maps OpenAI events to our RealtimeProviderEventMap:
   *   - connected → 'connected' (already emitted in connect())
   *   - disconnected → 'disconnected'
   *   - error → 'error'
   *   - conversation.item.input_audio_transcription.completed → 'transcript'
   *   - response.audio.delta → 'audio'
   */
  private wireEvents(): void {
    if (!this.client) {
      return;
    }

    // Connection events
    this.client.on('connected', () => {
      this.emitter.emit('connected');
    });

    this.client.on('disconnected', ((reason?: string) => {
      this.connected = false;
      this.pendingSpokes.clear();
      this.emitter.emit('disconnected', reason);
    }) as (...args: unknown[]) => void);

    this.client.on('error', (error: unknown) => {
      const err = error instanceof Error ? error : new Error(String(error));
      this.emitter.emit('error', err);
    });

    // Transcript events — STT results from audio input
    this.client.on(
      'conversation.item.input_audio_transcription.completed',
      ((event: unknown) => {
        // Runtime guard for event shape
        if (
          typeof event === 'object' &&
          event !== null &&
          'transcript' in event &&
          typeof (event as { transcript: unknown }).transcript === 'string'
        ) {
          const e = event as { transcript: string };
          const segment: TranscriptSegment = {
            speaker: 'user',
            text: e.transcript,
            timestamp: new Date().toISOString(),
          };
          this.emitter.emit('transcript', segment);
        }
      }) as (...args: unknown[]) => void
    );

    // Audio output events — TTS audio chunks
    this.client.on('response.audio.delta', ((event: unknown) => {
      // Runtime guard for event shape
      if (
        typeof event === 'object' &&
        event !== null &&
        'delta' in event &&
        typeof (event as { delta: unknown }).delta === 'string'
      ) {
        const e = event as { delta: string };
        // Decode base64 delta to Buffer
        const audioData = Buffer.from(e.delta, 'base64');
        const chunk: AudioChunk = {
          data: audioData,
          sampleRate: 24000,
          channels: 1,
        };
        this.emitter.emit('audio', chunk);
      }
    }) as (...args: unknown[]) => void);

    // Track completed responses — remove from pending
    this.client.on('response.done', ((event: unknown) => {
      if (
        typeof event === 'object' &&
        event !== null &&
        'response' in event
      ) {
        const e = event as { response?: { id?: string } };
        if (e.response?.id) {
          this.pendingSpokes.delete(e.response.id);
        }
      }
    }) as (...args: unknown[]) => void);
  }

  /**
   * Configure the realtime session with voice and audio settings.
   *
   * Called once after connect() — all values from config, nothing hardcoded.
   */
  private async configureSession(): Promise<void> {
    if (!this.client) {
      return;
    }

    // Update session configuration — all values from config
    await this.client.updateSession({
      modalities: ['text', 'audio'],
      instructions: this.config.instructions,
      voice: this.config.voice,
      input_audio_format: 'pcm16',
      output_audio_format: 'pcm16',
      input_audio_transcription: {
        model: this.config.transcriptionModel,
      },
      turn_detection: {
        type: 'server_vad',
        threshold: this.config.vad.threshold,
        prefix_padding_ms: this.config.vad.prefixPaddingMs,
        silence_duration_ms: this.config.vad.silenceDurationMs,
      },
    });
  }
}

/**
 * Factory function for creating OpenAI Realtime provider.
 *
 * @param config - Provider configuration (API key, model, voice, VAD, etc.)
 * @returns Configured RealtimeProvider instance
 */
export function createOpenAIRealtimeProvider(
  config: OpenAIRealtimeConfig
): RealtimeProvider {
  return new OpenAIRealtimeProvider(config);
}
