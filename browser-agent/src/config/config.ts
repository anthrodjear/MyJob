import * as fs from 'fs';
import * as path from 'path';
import { load, YAMLException, FAILSAFE_SCHEMA } from 'js-yaml';
import { z } from 'zod';

/**
 * Custom error type for configuration loading failures.
 * Wraps underlying errors (YAMLException, file system errors, validation errors)
 * with context about what failed.
 */
export class ConfigError extends Error {
  constructor(message: string, public readonly cause?: unknown) {
    super(message);
    this.name = 'ConfigError';
  }
}

// ----- Zod Schemas (single source of truth for config shape) -----

const ProviderConfigSchema = z.object({
  provider: z.string().min(1),
  model: z.string().min(1),
});

const LLMConfigSchema = z.object({
  primary: ProviderConfigSchema,
  local: ProviderConfigSchema.extend({
    baseUrl: z.string().url().optional(),
  }),
  embeddings: ProviderConfigSchema,
});

const PromptPairSchema = z.object({
  system: z.string().min(1),
  user: z.string().min(1),
});

const PromptsConfigSchema = z.object({
  scoring: PromptPairSchema,
  email_classifier: PromptPairSchema,
  cover_letter: PromptPairSchema,
  resume_tailor: PromptPairSchema,
  interview_prep: PromptPairSchema,
  job_extraction: PromptPairSchema,
  form_understanding: PromptPairSchema,
  resume_generation: PromptPairSchema,
});

const ApplicationConfigSchema = z.object({
  approval_tiers: z.object({
    auto_apply: z.object({
      min_score: z.number().min(0).max(100),
      action: z.string().min(1),
      notify: z.boolean(),
    }),
    review: z.object({
      min_score: z.number().min(0).max(100),
      max_score: z.number().min(0).max(100),
      action: z.string().min(1),
    }),
    reject: z.object({
      max_score: z.number().min(0).max(100),
      action: z.string().min(1),
      log: z.boolean(),
    }),
  }),
  auto_generate: z.object({
    resume: z.boolean(),
    cover_letter: z.boolean(),
  }),
  resume: z.object({
    engine: z.string().min(1),
    template_dir: z.string().min(1),
  }),
  cover_letter: z.object({
    engine: z.string().min(1),
    template_dir: z.string().min(1),
    max_length: z.number().positive(),
  }),
});

const QueueConfigSchema = z.object({
  redis_url: z.string().min(1),
  concurrency: z.number().int().positive(),
  retryAttempts: z.number().int().min(0),
});

const VoiceVADConfigSchema = z.object({
  threshold: z.number().min(0).max(1).default(0.5),
  prefix_padding_ms: z.number().int().min(0).default(300),
  silence_duration_ms: z.number().int().min(0).default(500),
});

const OpenAIRealtimeConfigSchema = z.object({
  voice: z.string().min(1).default('alloy'),
  transcription_model: z.string().min(1).default('whisper-1'),
  instructions: z.string().default(''),
  vad: VoiceVADConfigSchema.default({}),
});

const ElevenLabsTTSConfigSchema = z.object({
  voice_id: z.string().min(1).default('21m00Tcm4TlvDq8ikWAM'),
  model_id: z.string().min(1).default('eleven_multilingual_v2'),
  stability: z.number().min(0).max(1).default(0.5),
  similarity_boost: z.number().min(0).max(1).default(0.75),
  output_format: z.string().min(1).default('pcm_24000'),
});

const WhisperSTTConfigSchema = z.object({
  model_id: z.string().min(1).default('whisper-1'),
  language: z.string().min(2).default('en'),
  prompt: z.string().default(''),
});

const LocalWhisperConfigSchema = z.object({
  binary: z.string().min(1).default('whisper'),
  model: z.string().min(1).default('base'),
  language: z.string().min(2).default('en'),
  backend: z.enum(['whisper', 'faster-whisper']).default('whisper'),
  timeout_ms: z.number().int().min(1000).default(60_000),
  output_dir: z.string().optional(),
});

const LocalPiperConfigSchema = z.object({
  binary: z.string().min(1).default('piper'),
  model: z.string().min(1).default(''),
  sample_rate: z.number().int().min(8000).default(22_050),
  length_scale: z.number().min(0.1).max(3.0).default(1.0),
  noise_scale: z.number().min(0).max(1).default(0.667),
  noise_w: z.number().min(0).max(1).default(0.8),
  sentence_silence: z.number().min(0).max(5).default(0.2),
  speaker_id: z.number().int().min(0).optional(),
  args: z.array(z.string()).optional(),
  timeout_ms: z.number().int().min(1000).default(30_000),
});

const LocalKokoroConfigSchema = z.object({
  python: z.string().min(1).default('python3'),
  model_path: z.string().min(1).default(''),
  voices_path: z.string().min(1).default(''),
  voice: z.string().min(1).default('af_nicole'),
  language: z.string().min(2).default('en-us'),
  speed: z.number().min(0.5).max(2.0).default(1.0),
  sample_rate: z.number().int().min(8000).default(24_000),
  timeout_ms: z.number().int().min(1000).default(30_000),
});

const VoiceConfigSchema = z.object({
  provider: z.string().min(1),
  model: z.string().min(1),
  livekit: z.object({
    url: z.string().min(1),
    api_key: z.string().min(1),
    api_secret: z.string().min(1),
  }),
  openai_realtime: OpenAIRealtimeConfigSchema.default({}),
  elevenlabs: ElevenLabsTTSConfigSchema.default({}),
  whisper: WhisperSTTConfigSchema.default({}),
  local_whisper: LocalWhisperConfigSchema.default({}),
  local_piper: LocalPiperConfigSchema.default({}),
  local_kokoro: LocalKokoroConfigSchema.default({}),
  local_tts: z.enum(['piper', 'kokoro']).default('piper'),
});

const MemoryConfigSchema = z.object({
  max_recent_segments: z.number().int().min(1).default(50),
  keep_after_summarize: z.number().int().min(1).default(10),
  max_accumulated_items: z.number().int().min(1).default(100),
  max_summary_length: z.number().int().min(100).default(3000),
});

const RetrieverConfigSchema = z.object({
  timeout_ms: z.number().int().min(100).default(5000),
  retries: z.number().int().min(0).default(2),
  max_content_length: z.number().int().min(100).default(10000),
});

const PromptBudgetSchema = z.object({
  system: z.number().int().min(100).default(3000),
  retrieval: z.number().int().min(100).default(6000),
  summary: z.number().int().min(100).default(2000),
  transcript: z.number().int().min(100).default(4000),
  question: z.number().int().min(100).default(1000),
});

const MemoryCeilingsSchema = z.object({
  summary: z.number().int().min(100).default(1500),
  recent_transcript: z.number().int().min(100).default(2500),
  facts: z.number().int().min(100).default(800),
  covered_topics: z.number().int().min(100).default(500),
  questions_asked: z.number().int().min(100).default(500),
});

const LLMConfigRuntimeSchema = z.object({
  timeout_ms: z.number().int().min(1000).default(30000),
  retries: z.number().int().min(0).default(2),
});

const ResponderConfigSchema = z.object({
  llm: LLMConfigRuntimeSchema.default({}),
  prompt_budget: PromptBudgetSchema.default({}),
  memory_ceilings: MemoryCeilingsSchema.default({}),
  min_salvageable_length: z.number().int().min(1).default(10),
});

const PlannerConfigSchema = z.object({
  min_substantive_length: z.number().int().min(1).default(5),
  max_filler_ratio: z.number().min(0).max(1).default(0.6),
  min_content_words: z.number().int().min(1).default(2),
  duplicate_threshold: z.number().min(0).max(1).default(0.5),
});

const InterviewConfigSchema = z.object({
  memory: MemoryConfigSchema.default({}),
  retriever: RetrieverConfigSchema.default({}),
  responder: ResponderConfigSchema.default({}),
  planner: PlannerConfigSchema.default({}),
});

const EmailConfigSchema = z.object({
  provider: z.string().min(1),
  check_interval: z.string().min(1),
  folders: z.array(z.string()).min(1),
});

const ConfigSchema = z.object({
  application: ApplicationConfigSchema,
  queue: QueueConfigSchema,
  llm: LLMConfigSchema,
  voice: VoiceConfigSchema,
  interview: InterviewConfigSchema,
  email: EmailConfigSchema,
  prompts: PromptsConfigSchema,
});

// ----- Type exports (derived from Zod schemas) -----

export type LLMConfig = z.infer<typeof LLMConfigSchema>;
export type PromptPair = z.infer<typeof PromptPairSchema>;
export type PromptsConfig = z.infer<typeof PromptsConfigSchema>;
export type ApplicationConfig = z.infer<typeof ApplicationConfigSchema>;
export type QueueConfig = z.infer<typeof QueueConfigSchema>;
export type VoiceConfig = z.infer<typeof VoiceConfigSchema>;
export type InterviewConfig = z.infer<typeof InterviewConfigSchema>;
export type MemoryConfig = z.infer<typeof MemoryConfigSchema>;
export type RetrieverConfig = z.infer<typeof RetrieverConfigSchema>;
export type ResponderConfig = z.infer<typeof ResponderConfigSchema>;
export type PlannerConfig = z.infer<typeof PlannerConfigSchema>;
export type EmailConfig = z.infer<typeof EmailConfigSchema>;
export type Config = z.infer<typeof ConfigSchema>;

// ----- Loader -----

let configCache: Config | null = null;

/**
 * Apply environment variable overrides on top of parsed YAML config.
 * Critical values can be overridden without editing the YAML file.
 */
function applyEnvOverrides(config: Config): Config {
  const override = <T>(envVar: string | undefined, value: T): T =>
    envVar !== undefined && envVar !== '' ? (envVar as T) : value;

  return {
    ...config,
    queue: {
      ...config.queue,
      redis_url: override(process.env.REDIS_URL, config.queue.redis_url),
    },
    voice: {
      ...config.voice,
      livekit: {
        ...config.voice.livekit,
        api_key: override(process.env.LIVEKIT_API_KEY, config.voice.livekit.api_key),
        api_secret: override(process.env.LIVEKIT_API_SECRET, config.voice.livekit.api_secret),
      },
    },
    llm: {
      ...config.llm,
      local: {
        ...config.llm.local,
        baseUrl: override(process.env.OLLAMA_BASE_URL, config.llm.local.baseUrl),
      },
    },
  };
}

/**
 * Load and validate the application configuration from a YAML file.
 * Throws ConfigError on file read errors, YAML parse errors, or schema validation failures.
 */
export function loadConfig(configPath?: string): Config {
  if (configCache) {
    return configCache;
  }

  const filePath = configPath || path.resolve(process.cwd(), 'config', 'application.yaml');

  // 1. Read file
  let fileContents: string;
  try {
    fileContents = fs.readFileSync(filePath, 'utf8');
  } catch (e) {
    const code = (e as NodeJS.ErrnoException).code;
    if (code === 'ENOENT') {
      throw new ConfigError(`Config file not found: ${filePath}`, e);
    }
    throw new ConfigError(`Failed to read config file: ${filePath}`, e);
  }

  // 2. Parse YAML (with filename for better error messages, FAILSAFE_SCHEMA to prevent type coercion)
  let parsed: unknown;
  try {
    parsed = load(fileContents, {
      filename: path.basename(filePath),
      schema: FAILSAFE_SCHEMA,
    });
  } catch (e) {
    if (e instanceof YAMLException) {
      const line = e.mark?.line !== undefined ? e.mark.line + 1 : '?';
      const col = e.mark?.column !== undefined ? e.mark.column + 1 : '?';
      throw new ConfigError(
        `Invalid YAML in ${path.basename(filePath)}: ${e.reason} at line ${line}, col ${col}`,
        e,
      );
    }
    throw new ConfigError('Unknown YAML parsing error', e);
  }

  // 3. Validate against schema
  const result = ConfigSchema.safeParse(parsed);
  if (!result.success) {
    const issues = result.error.issues
      .map(i => `  - ${i.path.join('.')}: ${i.message}`)
      .join('\n');
    throw new ConfigError(`Config validation failed:\n${issues}`, result.error);
  }

  // 4. Apply environment variable overrides
  const config = applyEnvOverrides(result.data);
  configCache = config;
  return config;
}

/**
 * Clear the config cache. Useful for tests that need to reload config.
 */
export function clearConfigCache(): void {
  configCache = null;
}

// ----- Accessor helpers -----

export function getPrompts(): PromptsConfig {
  return loadConfig().prompts;
}

export function getLLMConfig(): LLMConfig {
  return loadConfig().llm;
}

export function getApplicationConfig(): ApplicationConfig {
  return loadConfig().application;
}