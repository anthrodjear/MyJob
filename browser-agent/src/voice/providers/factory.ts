/**
 * Provider factory for creating STT, TTS, and Realtime provider instances.
 *
 * Reads provider selection from the `voice:` section of application config
 * and creates the appropriate provider instances. This is the ONLY place
 * that knows about all available providers.
 *
 * Provider selection:
 *   voice.provider = 'openai_realtime' → uses openai_realtime sub-config
 *   voice.provider = 'elevenlabs'      → uses elevenlabs + whisper sub-configs
 *   voice.provider = 'local'           → uses local_whisper + local_piper/kokoro sub-configs
 *
 * Usage:
 *   import { createProviders } from './providers/factory';
 *   const { stt, tts, realtime } = createProviders(config.voice);
 */

import { logger } from '../../utils/logger.js';
import type { STTProvider, TTSProvider, RealtimeProvider } from '../types.js';
import type { OpenAIRealtimeConfig } from './openai-realtime.js';
import type { ElevenLabsTTSConfig, WhisperSTTConfig } from './elevenlabs.js';
import type { LocalWhisperConfig } from './local-whisper.js';
import type { LocalPiperConfig } from './local-piper.js';
import type { LocalKokoroConfig } from './local-kokoro.js';

const log = logger.child({ component: 'VoiceProviderFactory' });

// ----- Unified Voice Config Type -----

/** The `voice:` section from application.yaml (snake_case keys matching YAML). */
export interface VoiceProviderConfig {
  /** Provider selection: 'openai_realtime', 'elevenlabs', or 'local' */
  provider: string;
  /** Model identifier */
  model: string;
  /** LiveKit connection config */
  livekit: {
    url: string;
    api_key: string;
    api_secret: string;
  };
  /** OpenAI Realtime sub-config (used when provider='openai_realtime') */
  openai_realtime?: {
    voice?: string;
    transcription_model?: string;
    instructions?: string;
    vad?: {
      threshold?: number;
      prefix_padding_ms?: number;
      silence_duration_ms?: number;
    };
  };
  /** ElevenLabs TTS sub-config (used when provider='elevenlabs') */
  elevenlabs?: {
    voice_id?: string;
    model_id?: string;
    stability?: number;
    similarity_boost?: number;
    output_format?: string;
  };
  /** Whisper STT sub-config (used when provider='elevenlabs') */
  whisper?: {
    model_id?: string;
    language?: string;
    prompt?: string;
  };
  /** Local Whisper STT sub-config (used when provider='local') */
  local_whisper?: {
    binary?: string;
    model?: string;
    language?: string;
    backend?: string;
    timeout_ms?: number;
    output_dir?: string;
  };
  /** Local Piper TTS sub-config (used when provider='local', local_tts='piper') */
  local_piper?: {
    binary?: string;
    model?: string;
    sample_rate?: number;
    length_scale?: number;
    noise_scale?: number;
    noise_w?: number;
    sentence_silence?: number;
    speaker_id?: number;
    args?: string[];
    timeout_ms?: number;
  };
  /** Local Kokoro TTS sub-config (used when provider='local', local_tts='kokoro') */
  local_kokoro?: {
    python?: string;
    model_path?: string;
    voices_path?: string;
    voice?: string;
    language?: string;
    speed?: number;
    sample_rate?: number;
    timeout_ms?: number;
  };
  /** Which local TTS engine to use when provider='local' — 'piper' | 'kokoro' */
  local_tts?: 'piper' | 'kokoro';
}

// ----- Provider Creation -----

/**
 * Create an STT provider based on voice config.
 *
 * Maps:
 *   'openai_realtime' → throws (use createRealtimeProvider instead)
 *   'elevenlabs'      → ElevenLabs WhisperSTT
 *   'local'           → LocalWhisperSTT
 *
 * @param config - Full voice config from application.yaml
 * @returns Configured STTProvider instance
 * @throws Error if provider is unknown
 */
export async function createSTTProvider(config: VoiceProviderConfig): Promise<STTProvider> {
  switch (config.provider) {
    case 'openai_realtime': {
      log.error({ provider: config.provider }, 'OpenAI Realtime handles STT internally — use createRealtimeProvider()');
      throw new Error(
        'OpenAI Realtime provider handles STT internally — use createRealtimeProvider() instead'
      );
    }
    case 'elevenlabs': {
      const apiKey = process.env['ELEVENLABS_API_KEY'];
      if (!apiKey) {
        log.error({ env: 'ELEVENLABS_API_KEY' }, 'Environment variable is not set');
        throw new Error('ELEVENLABS_API_KEY is required for elevenlabs STT provider');
      }
      const { createWhisperSTT } = await import('./elevenlabs.js');
      const cfg: WhisperSTTConfig = {
        apiKey,
        modelId: config.whisper?.model_id ?? 'whisper-1',
        language: config.whisper?.language ?? 'en',
        prompt: config.whisper?.prompt ?? '',
      };
      return createWhisperSTT(cfg);
    }
    case 'local': {
      const { createLocalWhisperSTT } = await import('./local-whisper.js');
      const cfg: LocalWhisperConfig = {
        binary: config.local_whisper?.binary ?? 'whisper',
        model: config.local_whisper?.model ?? 'base',
        language: config.local_whisper?.language ?? 'en',
        backend: (config.local_whisper?.backend as 'whisper' | 'faster-whisper') ?? 'whisper',
        timeoutMs: config.local_whisper?.timeout_ms ?? 60_000,
        outputDir: config.local_whisper?.output_dir,
      };
      return createLocalWhisperSTT(cfg);
    }
    default:
      log.error({ provider: config.provider }, 'Unknown voice provider');
      throw new Error(`Unknown voice provider: ${config.provider}`);
  }
}

/**
 * Create a TTS provider based on voice config.
 *
 * Maps:
 *   'openai_realtime' → throws (use createRealtimeProvider instead)
 *   'elevenlabs'      → ElevenLabsTTS
 *   'local' + 'piper' → LocalPiperTTS
 *   'local' + 'kokoro' → LocalKokoroTTS
 *
 * @param config - Full voice config from application.yaml
 * @returns Configured TTSProvider instance
 * @throws Error if provider is unknown
 */
export async function createTTSProvider(config: VoiceProviderConfig): Promise<TTSProvider> {
  switch (config.provider) {
    case 'openai_realtime': {
      log.error({ provider: config.provider }, 'OpenAI Realtime handles TTS internally — use createRealtimeProvider()');
      throw new Error(
        'OpenAI Realtime provider handles TTS internally — use createRealtimeProvider() instead'
      );
    }
    case 'elevenlabs': {
      const apiKey = process.env['ELEVENLABS_API_KEY'];
      if (!apiKey) {
        log.error({ env: 'ELEVENLABS_API_KEY' }, 'Environment variable is not set');
        throw new Error('ELEVENLABS_API_KEY is required for elevenlabs TTS provider');
      }
      const { createElevenLabsTTS } = await import('./elevenlabs.js');
      const cfg: ElevenLabsTTSConfig = {
        apiKey,
        voiceId: config.elevenlabs?.voice_id ?? '21m00Tcm4TlvDq8ikWAM',
        modelId: config.elevenlabs?.model_id ?? 'eleven_multilingual_v2',
        stability: config.elevenlabs?.stability ?? 0.5,
        similarityBoost: config.elevenlabs?.similarity_boost ?? 0.75,
        outputFormat: config.elevenlabs?.output_format ?? 'pcm_24000',
      };
      return createElevenLabsTTS(cfg);
    }
    case 'local': {
      const localTts = config.local_tts ?? 'piper';
      if (localTts !== 'piper' && localTts !== 'kokoro') {
        log.warn({ localTts, fallback: 'piper' }, 'Unknown local_tts value — falling back to piper');
      }
      if (localTts === 'kokoro') {
        const { createLocalKokoroTTS } = await import('./local-kokoro.js');
        const cfg: LocalKokoroConfig = {
          python: config.local_kokoro?.python ?? 'python3',
          modelPath: config.local_kokoro?.model_path ?? '',
          voicesPath: config.local_kokoro?.voices_path ?? '',
          voice: config.local_kokoro?.voice ?? 'af_nicole',
          language: config.local_kokoro?.language ?? 'en-us',
          speed: config.local_kokoro?.speed ?? 1.0,
          sampleRate: config.local_kokoro?.sample_rate ?? 24_000,
          timeoutMs: config.local_kokoro?.timeout_ms ?? 30_000,
        };
        if (!cfg.modelPath || !cfg.voicesPath) {
          log.error({ modelPath: cfg.modelPath, voicesPath: cfg.voicesPath }, 'Kokoro model_path and voices_path are required');
          throw new Error('local_kokoro.model_path and local_kokoro.voices_path are required in voice config');
        }
        return createLocalKokoroTTS(cfg);
      }
      // Default to piper
      const { createLocalPiperTTS } = await import('./local-piper.js');
      const cfg: LocalPiperConfig = {
        binary: config.local_piper?.binary ?? 'piper',
        model: config.local_piper?.model ?? '',
        sampleRate: config.local_piper?.sample_rate ?? 22_050,
        lengthScale: config.local_piper?.length_scale ?? 1.0,
        noiseScale: config.local_piper?.noise_scale ?? 0.667,
        noiseW: config.local_piper?.noise_w ?? 0.8,
        sentenceSilence: config.local_piper?.sentence_silence ?? 0.2,
        speakerId: config.local_piper?.speaker_id,
        args: config.local_piper?.args,
        timeoutMs: config.local_piper?.timeout_ms ?? 30_000,
      };
      if (!cfg.model) {
        log.error({ field: 'local_piper.model' }, 'Local Piper model path is required');
        throw new Error('local_piper.model is required in voice.local_piper config');
      }
      return createLocalPiperTTS(cfg);
    }
    default:
      log.error({ provider: config.provider }, 'Unknown voice provider');
      throw new Error(`Unknown voice provider: ${config.provider}`);
  }
}

/**
 * Create a Realtime provider based on voice config.
 *
 * Only supports 'openai_realtime' provider. Other providers don't have
 * a realtime WebSocket API and should use separate STT + TTS.
 *
 * @param config - Full voice config from application.yaml
 * @returns Configured RealtimeProvider instance
 * @throws Error if provider is not 'openai_realtime'
 */
export async function createRealtimeProvider(config: VoiceProviderConfig): Promise<RealtimeProvider> {
  if (config.provider !== 'openai_realtime') {
    log.error({ provider: config.provider }, 'Realtime provider only available for openai_realtime');
    throw new Error(
      `Realtime provider only available for 'openai_realtime' — got '${config.provider}'`
    );
  }

  const apiKey = process.env['OPENAI_API_KEY'];
  if (!apiKey) {
    log.error({ env: 'OPENAI_API_KEY' }, 'Environment variable is not set');
    throw new Error('OPENAI_API_KEY is required for openai_realtime provider');
  }

  const { createOpenAIRealtimeProvider } = await import('./openai-realtime.js');
  const vadConfig = config.openai_realtime?.vad;
  const cfg: OpenAIRealtimeConfig = {
    apiKey,
    model: config.model,
    voice: config.openai_realtime?.voice ?? 'alloy',
    transcriptionModel: config.openai_realtime?.transcription_model ?? 'whisper-1',
    instructions: config.openai_realtime?.instructions ?? '',
    vad: {
      threshold: vadConfig?.threshold ?? 0.5,
      prefixPaddingMs: vadConfig?.prefix_padding_ms ?? 300,
      silenceDurationMs: vadConfig?.silence_duration_ms ?? 500,
    },
  };
  return createOpenAIRealtimeProvider(cfg);
}

/**
 * Create all providers from a single voice config.
 *
 * Convenience factory that creates STT, TTS, and optionally Realtime
 * providers from the voice section of application.yaml.
 *
 * @param config - The `voice:` section from application.yaml
 * @returns Object with stt, tts, and optionally realtime providers
 */
export async function createProviders(config: VoiceProviderConfig): Promise<{
  stt?: STTProvider;
  tts?: TTSProvider;
  realtime?: RealtimeProvider;
}> {
  const isRealtime = config.provider === 'openai_realtime';

  return {
    stt: isRealtime ? undefined : await createSTTProvider(config),
    tts: isRealtime ? undefined : await createTTSProvider(config),
    realtime: isRealtime ? await createRealtimeProvider(config) : undefined,
  };
}
