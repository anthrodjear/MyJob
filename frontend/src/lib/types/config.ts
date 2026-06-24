/**
 * System configuration types for the admin settings UI.
 *
 * Mirrors the backend EffectiveConfig structure returned by GET /api/v1/system/config.
 * Sources map tracks which layer (yaml/env/db) produced each value.
 */

/** Origin layer of a configuration value. */
export type ConfigSource = "default" | "yaml" | "env" | "db";

/** Scoring strategy mode. */
export type ScoringMode = "heuristic" | "llm" | "hybrid";

/** Scoring weights that sum to 1.0. */
export interface ScoringWeights {
  Skill: number;
  Experience: number;
  Location: number;
  Salary: number;
  Description: number;
}

/** Scoring thresholds, weights, and mode. */
export interface ScoringSection {
  auto_threshold: number;
  review_threshold: number;
  mode: ScoringMode;
  hybrid_reject_margin: number;
  weights: ScoringWeights;
}

/** Single LLM provider configuration. */
export interface LLMProviderSection {
  provider: string;
  model: string;
}

/** LLM provider configurations. */
export interface LLMSection {
  primary: LLMProviderSection;
  local: LLMProviderSection;
  embeddings: LLMProviderSection;
}

/** LiveKit connection parameters (secrets omitted). */
export interface LiveKitSection {
  url: string;
  api_key: string;
}

/** Voice provider and LiveKit configuration. */
export interface VoiceSection {
  provider: string;
  model: string;
  livekit: LiveKitSection;
  settings: Record<string, string>;
}

/** Single approval tier definition. */
export interface ApprovalTierDef {
  min_score: number;
  max_score?: number;
  action: string;
  notify?: boolean;
  log?: boolean;
}

/** Auto/review/reject tier definitions. */
export interface ApprovalTiersSection {
  auto_apply: ApprovalTierDef;
  review: ApprovalTierDef;
  reject: ApprovalTierDef;
}

/** Resume generation engine and template settings. */
export interface ResumeConfigSection {
  engine: string;
  template_dir: string;
}

/** Cover letter generation engine and template settings. */
export interface CoverLetterConfigSection {
  engine: string;
  template_dir: string;
  max_length: number;
}

/** Async queue processing settings. */
export interface QueueSection {
  concurrency: number;
  retry_attempts: number;
}

/** Automatic generation toggles. */
export interface AutoGenerateSection {
  resume: boolean;
  cover_letter: boolean;
}

/** Queue and auto-generation settings. */
export interface AutomationSection {
  queue: QueueSection;
  auto_generate: AutoGenerateSection;
}

/** Transcript window and eviction settings. */
export interface InterviewMemory {
  max_recent_segments: number;
  keep_after_summarize: number;
}

/** LLM timeout and retry configuration. */
export interface LLMTimeout {
  timeout_ms: number;
  retries: number;
}

/** LLM settings for the interview responder. */
export interface InterviewResponder {
  llm: LLMTimeout;
}

/** Decision thresholds for the interview planner. */
export interface InterviewPlanner {
  duplicate_threshold: number;
  min_substantive_length: number;
}

/** Interview agent runtime settings. */
export interface InterviewSection {
  memory: InterviewMemory;
  responder: InterviewResponder;
  planner: InterviewPlanner;
}

/** Email polling interval and folder configuration. */
export interface EmailSection {
  provider: string;
  check_interval: string;
  folders: string[];
}

/** API rate limit settings. */
export interface RateLimitsSection {
  rpm: number;
  burst: number;
}

/** Integration connection status. */
export type IntegrationStatusType = "connected" | "disconnected" | "error";

/** Connection health and optional URL for a service. */
export interface IntegrationStatus {
  status: IntegrationStatusType;
  url?: string;
}

/** AI provider model and connection status. */
export interface AIProviderInfo {
  model: string;
  status: IntegrationStatusType;
}

/** Connection health for external services. */
export interface IntegrationsSection {
  livekit: IntegrationStatus;
  email: IntegrationStatus;
  ai_providers: Record<string, AIProviderInfo>;
}

/** Fully resolved configuration tree returned by GET /api/v1/system/config. */
export interface EffectiveConfig {
  scoring: ScoringSection;
  llm: LLMSection;
  voice: VoiceSection;
  approval_tiers: ApprovalTiersSection;
  resume: ResumeConfigSection;
  cover_letter: CoverLetterConfigSection;
  automation: AutomationSection;
  interview: InterviewSection;
  email: EmailSection;
  rate_limits: RateLimitsSection;
  integrations: IntegrationsSection;
  sources: Record<string, ConfigSource>;
}

/** API response wrapper for GET /api/v1/system/config. */
export interface SystemConfigResponse {
  config: EffectiveConfig;
  version?: string;
}

/** Request to set a configuration override. */
export interface SetOverrideRequest {
  key: string;
  value: unknown;
}

/** Response after setting a configuration override. */
export interface SetOverrideResponse {
  message: string;
  key: string;
}

/** Response after deleting a configuration override. */
export interface DeleteOverrideResponse {
  message: string;
  key: string;
}
