/**
 * Type declarations for @openai/realtime-api-beta.
 *
 * This package is ESM-only and may not ship with TypeScript types.
 * These declarations cover the methods used by openai-realtime.ts provider.
 *
 * @see https://github.com/openai/openai-realtime-api-beta
 */

declare module '@openai/realtime-api-beta' {
  export interface RealtimeClientConfig {
    apiKey: string;
    model: string;
  }

  export interface SessionConfig {
    modalities?: string[];
    instructions?: string;
    voice?: string;
    input_audio_format?: string;
    output_audio_format?: string;
    input_audio_transcription?: {
      model: string;
    };
    turn_detection?: {
      type: string;
      threshold?: number;
      prefix_padding_ms?: number;
      silence_duration_ms?: number;
    };
  }

  export interface ConversationItem {
    type: string;
    role?: string;
    content?: Array<{ type: string; text: string }>;
    id?: string;
  }

  export interface ResponseConfig {
    conversation_item_id?: string;
  }

  export class RealtimeClient {
    constructor(config: RealtimeClientConfig);

    /** Connect WebSocket to OpenAI Realtime API */
    connect(): Promise<void>;

    /** Close WebSocket connection */
    close(): Promise<void>;

    /** Send an event to the server */
    send(event: Record<string, unknown>): void;

    /** Send raw audio data to the server */
    sendAudio(audio: ArrayBuffer): void;

    /** Update session configuration */
    updateSession(config: SessionConfig): Promise<void>;

    /** Subscribe to events */
    on(event: string, handler: (...args: unknown[]) => void): void;

    /** Remove all event listeners */
    removeAllListeners(): void;
  }
}
