/**
 * Interview Session Factory — the ONLY place that reads config, env vars,
 * and creates concrete implementations.
 *
 * All DI wiring happens here. Session receives pre-built dependencies.
 */

import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { EventEmitter } from 'node:events';
import { logger } from '../utils/logger.js';
import type {
  LiveKitTransport,
  Transport,
  TransportEventMap,
  BrainConfig,
  AudioChunk,
  LiveKitConfig,
  VoiceSessionConfig,
  STTProviderName,
  TTSProviderName,
  RealtimeProviderName,
} from './types.js';
import { LiveKitTransportImpl } from './livekit.js';
import { createBrain } from './brain/index.js';
import { createAudioSegmentQueue } from './queue.js';
import { createInterviewSession, type SessionDeps } from './session.js';
import { createVAD } from './vad.js';
import { createSTTProvider, createTTSProvider, createRealtimeProvider, type VoiceProviderConfig } from './providers/factory.js';

const log = logger.child({ component: 'SessionFactory' });

// ----- Configuration types -----

/** YAML config shape for voice section. */
interface VoiceYamlConfig {
  provider?: string;
  livekit?: {
    url?: string;
    api_key?: string;
    api_secret?: string;
  };
  openai?: {
    model?: string;
    voice?: string;
  };
  elevenlabs?: {
    voice_id?: string;
    model_id?: string;
    speed?: number;
  };
  local?: {
    stt?: string;
    tts?: string;
    tts_model_dir?: string;
    transcribe_model?: string;
    whisper_binary?: string;
    whisper_model?: string;
    whisper_language?: string;
    piper_binary?: string;
    piper_model?: string;
    kokoro_model?: string;
    kokoro_voices?: string;
    kokoro_python?: string;
    kokoro_script?: string;
    sample_rate?: number;
    local_tts?: 'piper' | 'kokoro';
  };
  vad?: {
    silence_threshold_ms?: number;
    energy_threshold?: number;
    sample_rate?: number;
  };
  brain?: {
    planner?: { model?: string; temperature?: number };
    responder?: { model?: string; temperature?: number };
    max_tokens?: number;
    plan_interval?: number;
    backend_url?: string;
  };
  whisper?: {
    model_id?: string;
    language?: string;
    prompt?: string;
  };
}

/** Minimal YAML parser (avoids js-yaml dependency for just top-level keys). */
function parseVoiceConfig(yamlContent: string): VoiceYamlConfig {
  const config: VoiceYamlConfig = {};
  const lines = yamlContent.split('\n');
  let currentSection: string | null = null;

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;

    const indent = line.length - line.trimStart().length;

    if (indent === 0) {
      const colonIdx = trimmed.indexOf(':');
      if (colonIdx > 0) {
        currentSection = trimmed.slice(0, colonIdx).trim();
        const value = trimmed.slice(colonIdx + 1).trim();
        if (currentSection === 'provider' && value) {
          config.provider = value.replace(/['"]/g, '');
        }
      } else {
        currentSection = trimmed;
      }
    } else if (indent >= 2 && currentSection) {
      const colonIdx = trimmed.indexOf(':');
      if (colonIdx > 0) {
        const key = trimmed.slice(0, colonIdx).trim();
        const value = trimmed.slice(colonIdx + 1).trim();
        if (!value) {
          // Sub-section header
          if (!config[currentSection as keyof VoiceYamlConfig]) {
            (config as Record<string, unknown>)[currentSection] = {};
          }
          continue;
        }
        const parsed = value.replace(/['"]/g, '');
        const section = config[currentSection as keyof VoiceYamlConfig];
        if (typeof section === 'object' && section !== null) {
          const numVal = Number(parsed);
          (section as Record<string, unknown>)[key] = isNaN(numVal) ? parsed : numVal;
        }
      }
    }
  }

  return config;
}

// ----- Provider name helpers -----

function isSTTProvider(name: string): name is STTProviderName {
  return ['local-whisper', 'openai', 'deepgram'].includes(name);
}

function isTTSProvider(name: string): name is TTSProviderName {
  return ['local-piper', 'local-kokoro', 'elevenlabs'].includes(name);
}

function isRealtimeProvider(name: string): name is RealtimeProviderName {
  return ['openai_realtime'].includes(name);
}

// ----- Transport adapter -----

/**
 * Wraps LiveKitTransportImpl into the simplified Transport interface
 * that the session depends on. The session never sees LiveKitConfig.
 */
function createTransportAdapter(impl: LiveKitTransport, lkConfig: LiveKitConfig): Transport {
  const emitter = new EventEmitter();

  // Forward events from impl to adapter
  const forward = <K extends keyof TransportEventMap>(
    event: K,
    ...args: Parameters<TransportEventMap[K]>
  ) => {
    emitter.emit(event, ...args);
  };

  impl.on('connected', (roomName) => forward('connected', roomName));
  impl.on('disconnected', (reason) => forward('disconnected', reason));
  impl.on('audioReceived', (audio, participantId) => forward('audioReceived', audio, participantId));
  impl.on('participantJoined', (identity, kind) => forward('participantJoined', identity, kind));
  impl.on('participantLeft', (identity, reason) => forward('participantLeft', identity, reason));
  impl.on('error', (error) => forward('error', error));

  return {
    async connect(roomName: string, token: string): Promise<void> {
      await impl.connect(lkConfig, roomName, token);
    },
    disconnect: () => impl.disconnect(),
    publishAudio: (audio: AudioChunk) => impl.publishAudio(audio),
    on<K extends keyof TransportEventMap>(event: K, handler: TransportEventMap[K]): void {
      emitter.on(event, handler);
    },
    off<K extends keyof TransportEventMap>(event: K, handler: TransportEventMap[K]): void {
      emitter.off(event, handler);
    },
    get connected(): boolean {
      return impl.connected;
    },
  };
}

// ----- Config loading -----

function loadVoiceConfig(): VoiceYamlConfig {
  // 1. Try to load from YAML config file
  const configPath = process.env.VOICE_CONFIG_PATH
    ?? join(process.cwd(), 'config', 'voice.yaml');

  try {
    const content = readFileSync(configPath, 'utf-8');
    const config = parseVoiceConfig(content);
    log.info({ message: 'Loaded voice config from file', path: configPath });
    return config;
  } catch (err) {
    log.warn({ message: 'Could not load voice config file, using env vars only', path: configPath });
  }

  return {};
}

function getEnvOrConfig(
  config: VoiceYamlConfig,
  section: string,
  key: string,
  envKey: string,
  defaultValue?: string,
): string | undefined {
  // Env vars take precedence
  if (process.env[envKey]) return process.env[envKey];

  // Then YAML config
  const sectionObj = config[section as keyof VoiceYamlConfig];
  if (typeof sectionObj === 'object' && sectionObj !== null) {
    const val = (sectionObj as Record<string, unknown>)[key];
    if (typeof val === 'string') return val;
  }

  return defaultValue;
}

function getNumericEnvOrConfig(
  config: VoiceYamlConfig,
  section: string,
  key: string,
  envKey: string,
  defaultValue: number,
): number {
  const envVal = process.env[envKey];
  if (envVal) {
    const num = Number(envVal);
    if (!isNaN(num)) return num;
  }

  const sectionObj = config[section as keyof VoiceYamlConfig];
  if (typeof sectionObj === 'object' && sectionObj !== null) {
    const val = (sectionObj as Record<string, unknown>)[key];
    if (typeof val === 'number') return val;
  }

  return defaultValue;
}

// ----- Public factory -----

/**
 * Creates a fully-wired InterviewSession.
 *
 * Reads config from environment/YAML, creates all concrete implementations,
 * wires them together, and returns a session ready to use.
 *
 * @param overrides - Partial overrides for testing/customization
 */
export function createInterviewSessionFactory(overrides?: {
  config?: VoiceYamlConfig;
  lkConfig?: Partial<LiveKitConfig>;
  backendUrl?: string;
}): {
  session: ReturnType<typeof createInterviewSession>;
  config: VoiceYamlConfig;
} {
  const config = overrides?.config ?? loadVoiceConfig();

  // LiveKit config (env vars or YAML)
  const lkConfig: LiveKitConfig = {
    url: overrides?.lkConfig?.url
      ?? getEnvOrConfig(config, 'livekit', 'url', 'LIVEKIT_URL', 'ws://localhost:7880')!,
    apiKey: overrides?.lkConfig?.apiKey
      ?? getEnvOrConfig(config, 'livekit', 'api_key', 'LIVEKIT_API_KEY')!,
    apiSecret: overrides?.lkConfig?.apiSecret
      ?? getEnvOrConfig(config, 'livekit', 'api_secret', 'LIVEKIT_API_SECRET')!,
  };

  if (!lkConfig.apiKey || !lkConfig.apiSecret) {
    throw new Error(
      'LiveKit API key and secret are required. '
      + 'Set LIVEKIT_API_KEY and LIVEKIT_API_SECRET environment variables, '
      + 'or configure them in config/voice.yaml under livekit.',
    );
  }

  // Backend URL for context retrieval
  const backendUrl = overrides?.backendUrl
    ?? getEnvOrConfig(config, 'brain', 'backend_url', 'BACKEND_URL', 'http://localhost:8080')!;

  // Provider selection
  const provider = config.provider ?? 'openai_realtime';

  // VAD config
  const vadConfig = {
    silenceThresholdMs: getNumericEnvOrConfig(config, 'vad', 'silence_threshold_ms', 'VAD_SILENCE_THRESHOLD_MS', 1500),
    energyThreshold: getNumericEnvOrConfig(config, 'vad', 'energy_threshold', 'VAD_ENERGY_THRESHOLD', 0.02),
    sampleRate: getNumericEnvOrConfig(config, 'vad', 'sample_rate', 'VAD_SAMPLE_RATE', 16000),
  };

  log.info({
    message: 'Creating interview session',
    provider,
    backendUrl,
    livekitUrl: lkConfig.url,
  });

  // Convert YAML config to provider config with proper structure
  function toProviderConfig(cfg: VoiceYamlConfig): VoiceProviderConfig {
    return {
      provider: cfg.provider ?? 'openai_realtime',
      model: cfg.openai?.model ?? 'gpt-4o-realtime-preview',
      livekit: {
        url: cfg.livekit?.url ?? '',
        api_key: cfg.livekit?.api_key ?? process.env['LIVEKIT_API_KEY'] ?? '',
        api_secret: cfg.livekit?.api_secret ?? process.env['LIVEKIT_API_SECRET'] ?? '',
      },
      openai_realtime: cfg.openai,
      elevenlabs: cfg.elevenlabs,
      whisper: cfg.whisper,
      local_whisper: cfg.local?.stt === 'local-whisper' ? {
        binary: cfg.local?.whisper_binary,
        model: cfg.local?.whisper_model,
        language: cfg.local?.whisper_language,
        backend: cfg.local?.whisper_binary?.includes('faster') ? 'faster-whisper' : 'whisper',
        timeout_ms: 60_000,
        output_dir: undefined,
      } : undefined,
      local_piper: cfg.local?.tts === 'local-piper' || cfg.local?.local_tts === 'piper' ? {
        binary: cfg.local?.piper_binary,
        model: cfg.local?.piper_model,
        sample_rate: cfg.local?.sample_rate,
        length_scale: undefined,
        noise_scale: undefined,
        noise_w: undefined,
        sentence_silence: undefined,
        speaker_id: undefined,
        args: undefined,
        timeout_ms: undefined,
      } : undefined,
      local_kokoro: cfg.local?.tts === 'local-kokoro' || cfg.local?.local_tts === 'kokoro' ? {
        python: cfg.local?.kokoro_python,
        model_path: cfg.local?.kokoro_model,
        voices_path: cfg.local?.kokoro_voices,
        voice: undefined,
        language: undefined,
        speed: undefined,
        sample_rate: cfg.local?.sample_rate,
        timeout_ms: undefined,
      } : undefined,
      local_tts: cfg.local?.local_tts,
    };
  }

  const providerConfig = toProviderConfig(config);

  // Create transport (LiveKit)
  const livekitImpl = new LiveKitTransportImpl();
  const transport = createTransportAdapter(livekitImpl, lkConfig);

  // Create brain
  const brainConfig: BrainConfig = {
    memory: {
      max_recent_segments: getNumericEnvOrConfig(config, 'brain', 'max_recent_segments', 'BRAIN_MAX_RECENT_SEGMENTS', 30),
    },
    retriever: {
      request_timeout_ms: getNumericEnvOrConfig(config, 'brain', 'retriever_timeout_ms', 'BRAIN_RETRIEVER_TIMEOUT_MS', 10000),
    },
    responder: {
      model: config.brain?.responder?.model ?? 'qwen2.5:7b',
      prompt_budget: {
        system: 1500,
        retrieval: 2000,
        summary: 800,
        transcript: 3000,
        question: 500,
      },
    },
    planner: {},
  };

  const brain = createBrain(backendUrl, brainConfig);

  // Session deps
  let sessionDeps: SessionDeps;

  // Create providers based on config
  if (isRealtimeProvider(provider)) {
    // Realtime path (single provider handles STT+TTS)
    const realtimeProvider = createRealtimeProvider(providerConfig);

    sessionDeps = {
      transport,
      brain,
      realtime: realtimeProvider,
    };
  } else {
    // Separate STT/TTS path
    const sttName = config.local?.stt ?? 'local-whisper';
    const ttsName = config.local?.tts ?? 'local-piper';

    if (!isSTTProvider(sttName)) {
      throw new Error(`Invalid STT provider: ${sttName}. Expected: local-whisper, openai, deepgram`);
    }
    if (!isTTSProvider(ttsName)) {
      throw new Error(`Invalid TTS provider: ${ttsName}. Expected: local-piper, local-kokoro, elevenlabs`);
    }

    const vad = createVAD({
      energyThreshold: vadConfig.energyThreshold,
      silenceThresholdMs: vadConfig.silenceThresholdMs,
      sampleRate: vadConfig.sampleRate,
    });
    const stt = createSTTProvider(providerConfig);
    const tts = createTTSProvider(providerConfig);

    sessionDeps = {
      transport,
      brain,
      vad,
      stt,
      tts,
    };
  }

  // Create the session orchestrator
  const session = createInterviewSession(sessionDeps);

  return { session, config };
}

// ----- Re-exports -----
export type { SessionDeps } from './session.js';
export { createInterviewSession } from './session.js';
