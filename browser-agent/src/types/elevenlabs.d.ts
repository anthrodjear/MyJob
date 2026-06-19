/**
 * Type declarations for elevenlabs package.
 *
 * This package is ESM-only and may not ship with TypeScript types.
 * These declarations cover the methods used by elevenlabs.ts provider.
 *
 * @see https://github.com/elevenlabs/elevenlabs-js
 */

declare module 'elevenlabs' {
  export interface ElevenLabsClientConfig {
    apiKey: string;
  }

  export interface TextToSpeechConvertParams {
    text: string;
    model_id: string;
    output_format?: string;
    voice_settings?: {
      stability?: number;
      similarity_boost?: number;
    };
  }

  export interface ElevenLabsClient {
    textToSpeech: {
      convert: (
        voiceId: string,
        params: TextToSpeechConvertParams
      ) => Promise<ReadableStream<Uint8Array>>;
      stream: (
        voiceId: string,
        params: TextToSpeechConvertParams
      ) => Promise<ReadableStream<Uint8Array>>;
    };
  }

  export class ElevenLabsClient implements ElevenLabsClient {
    constructor(config: ElevenLabsClientConfig);
    textToSpeech: ElevenLabsClient['textToSpeech'];
  }
}
