/**
 * Voice module types — all interfaces for the Interview Agent.
 *
 * Architecture:
 *   Transport (LiveKit) → VAD → Speech (STT/TTS/Realtime) → Brain (planner/responder/memory/retrieval) → Session
 *
 * Key principle: Interview intelligence lives in brain/, not inside voice providers.
 * Providers are pure STT/TTS — they don't know about resumes, jobs, or interviews.
 */

// ----- Interview Modes -----

/** Assist Mode: user attends interview, agent provides real-time suggestions. */
export type InterviewMode = 'assist' | 'autonomous';

// ----- Session State -----

/** State machine for interview sessions. */
export type SessionState =
  | 'idle'
  | 'connecting'
  | 'reconnecting'
  | 'listening'
  | 'thinking'
  | 'speaking'
  | 'ended'
  | 'error';

// ----- Transcript -----

/** A single transcript segment from the conversation. */
export interface TranscriptSegment {
  /** Who said it: 'user' (candidate) | 'interviewer' | 'agent' (our system) */
  speaker: 'user' | 'interviewer' | 'agent';
  /** Unique identifier when multiple speakers share a role (panel interviews) */
  speakerId?: string;
  /** The spoken text */
  text: string;
  /** ISO 8601 timestamp — survives network boundaries and database storage */
  timestamp: string;
  /** Confidence score from STT (0-1), undefined if not available */
  confidence?: number;
}

// ----- Audio -----

/** Audio chunk from LiveKit transport. */
export interface AudioChunk {
  /** Raw PCM audio data — Buffer for Node, ArrayBuffer for browser */
  readonly data: Buffer | ArrayBuffer;
  /** Sample rate in Hz (e.g., 16000, 44100) */
  readonly sampleRate: number;
  /** Number of audio channels */
  readonly channels: number;
}

// ----- Voice Activity Detection (VAD) -----

/** Audio events emitted by VAD — signals speech boundaries. */
export type AudioEvent =
  | { type: 'speech_started'; timestamp: string }
  | { type: 'speech_ended'; timestamp: string; durationMs: number }
  | { type: 'silence_detected'; durationMs: number };

/** VAD event handler map — type-safe per event. */
export interface AudioEventMap {
  speech_started: (timestamp: string) => void;
  speech_ended: (timestamp: string, durationMs: number) => void;
  silence_detected: (durationMs: number) => void;
}

/**
 * Voice Activity Detection — detects when someone is speaking.
 *
 * Critical for interrupt-driven interviews:
 *   Interviewer starts talking → agent stops speaking
 *   Pause long enough → agent answers
 *
 * Lifecycle: initialize() → start/stop detection → destroy().
 */
export interface VoiceActivityDetector {
  readonly name: string;

  initialize(): Promise<void>;

  /**
   * Analyze an audio chunk and determine speech state.
   * Returns true if speech is detected (energy above threshold).
   */
  isSpeaking(audio: AudioChunk): boolean;

  /**
   * Get current speech state with debounce info.
   * Returns null if VAD hasn't seen enough audio yet.
   */
  getState(): SpeechState | null;

  /** Subscribe to speech boundary events (started/ended/silence). */
  on<K extends keyof AudioEventMap>(event: K, handler: AudioEventMap[K]): void;

  /** Remove event handler. */
  off<K extends keyof AudioEventMap>(event: K, handler: AudioEventMap[K]): void;

  /** Update sensitivity thresholds at runtime (e.g., for noisy environments). */
  updateConfig(config: VADConfig): void;

  destroy(): Promise<void>;
}

/** Current VAD speech state. */
export interface SpeechState {
  /** Whether speech is currently detected */
  isSpeaking: boolean;
  /** Duration of current speech segment in ms (0 if silence) */
  currentSegmentMs: number;
  /** Duration of current silence gap in ms (0 if speaking) */
  silenceGapMs: number;
  /** RMS energy level of last analyzed chunk (0-1) */
  energyLevel: number;
}

/** Configuration for VAD sensitivity. */
export interface VADConfig {
  /** Energy threshold for speech detection (0-1, default 0.02) */
  energyThreshold?: number;
  /** Minimum silence duration (ms) to consider speech ended (default 700) */
  silenceThresholdMs?: number;
  /** Minimum speech duration (ms) to trigger speech_started (default 200) */
  speechThresholdMs?: number;
  /** Sample rate expected by VAD (must match audio source) */
  sampleRate?: number;
}

// ----- STT (Speech-to-Text) -----

/** Speech-to-text provider interface. Lifecycle: initialize() → transcribe() → destroy(). */
export interface STTProvider {
  /** Provider name for logging/config */
  readonly name: string;

  /** Initialize the STT provider (connect to service, load models) */
  initialize(): Promise<void>;

  /**
   * Transcribe audio chunks to text.
   * Called continuously as audio arrives from LiveKit.
   */
  transcribe(audio: AudioChunk): Promise<TranscriptSegment | null>;

  /**
   * Streaming variant — yields partial results as audio is processed.
   * Optional: batch-only providers don't implement this.
   */
  transcribeStream?(audio: AudioChunk): AsyncIterable<TranscriptSegment>;

  /** Clean up resources */
  destroy(): Promise<void>;
}

// ----- TTS (Text-to-Speech) -----

/** Options for TTS synthesis. */
export interface TTSOptions {
  /** Voice ID or name (provider-specific) */
  voice?: string;
  /** Speech speed (0.5 = slow, 1.0 = normal, 2.0 = fast) */
  speed?: number;
  /** Language code (e.g., 'en', 'es') */
  language?: string;
}

/**
 * Text-to-speech provider interface. Lifecycle: initialize() → synthesize() → destroy().
 *
 * ⚠️ synthesize() returns an AsyncIterable — audio streams chunk-by-chunk.
 * Do NOT buffer the entire text before speaking. Real-time audio must stream.
 */
export interface TTSProvider {
  /** Provider name for logging/config */
  readonly name: string;

  /** Initialize the TTS provider */
  initialize(): Promise<void>;

  /**
   * Convert text to audio stream.
   * Yields audio chunks as they're ready — caller publishes each to LiveKit immediately.
   */
  synthesize(text: string, options?: TTSOptions): AsyncIterable<AudioChunk>;

  /** Clean up resources */
  destroy(): Promise<void>;
}

// ----- Realtime Provider (adapter for combined STT+TTS services) -----

/** Events emitted by a realtime provider — type-safe per event. */
export interface RealtimeProviderEventMap {
  connected: () => void;
  disconnected: (reason?: string) => void;
  transcript: (segment: TranscriptSegment) => void;
  audio: (audio: AudioChunk) => void;
  error: (error: Error) => void;
}

/**
 * Adapter for services that handle both STT and TTS in one connection.
 * Example: OpenAI Realtime API does STT + LLM + TTS in one WebSocket.
 *
 * This is NOT the primary architecture. Most providers use separate STT + TTS.
 * Use this only when the provider natively combines both.
 *
 * Lifecycle: connect() → sendAudio()/speak()/cancel() → disconnect() → destroy().
 */
export interface RealtimeProvider {
  /** Provider name for logging/config */
  readonly name: string;

  /** Connect to the provider's WebSocket/API */
  connect(): Promise<void>;

  /** Disconnect from the provider */
  disconnect(): Promise<void>;

  /** Send audio to the provider for processing */
  sendAudio(audio: AudioChunk): Promise<void>;

  /**
   * Text-based speech intervention.
   * Use for: system prompts, interrupting a speaker, cancelling ongoing audio loops.
   * When candidate cuts off the AI, call cancel() then speak() with revised text.
   */
  speak(text: string): Promise<void>;

  /**
   * Cancel ongoing audio output.
   * Use when: candidate interrupts, agent needs to revise response, error recovery.
   * After cancel(), the provider stops speaking and returns to listening state.
   */
  cancel(): Promise<void>;

  /** Subscribe to realtime provider events with type-safe callback. */
  on<K extends keyof RealtimeProviderEventMap>(event: K, handler: RealtimeProviderEventMap[K]): void;

  /** Remove event handler. */
  off<K extends keyof RealtimeProviderEventMap>(event: K, handler: RealtimeProviderEventMap[K]): void;

  /** Clean up all resources */
  destroy(): Promise<void>;
}

// ----- LiveKit Transport -----

/**
 * Configuration for LiveKit transport.
 * ⚠️ Contains secrets — never log this object directly. Use redactedConfig().
 */
export interface LiveKitConfig {
  /** LiveKit server URL (e.g., 'ws://localhost:7880') */
  url: string;
  /** API key for authentication */
  apiKey: string;
  /** API secret for token generation */
  apiSecret: string;
}

/** Redacted version of LiveKitConfig for safe logging. */
export type RedactedLiveKitConfig = Omit<LiveKitConfig, 'apiKey' | 'apiSecret'> & {
  apiKey: '***';
  apiSecret: '***';
};

/** Create a redacted copy of LiveKitConfig for safe logging. */
export function redactConfig(config: LiveKitConfig): RedactedLiveKitConfig {
  return { url: config.url, apiKey: '***', apiSecret: '***' };
}

/** Events emitted by the LiveKit transport — discriminated per event type. */
export type TransportEvent =
  | { type: 'connected'; roomName: string }
  | { type: 'disconnected'; reason?: string }
  | { type: 'audioReceived'; audio: AudioChunk; participantId: string }
  | { type: 'participantJoined'; identity: string; kind: string }
  | { type: 'participantLeft'; identity: string; reason?: string }
  | { type: 'error'; error: Error };

/** Transport event handler map — type-safe per event. */
export interface TransportEventMap {
  connected: (roomName: string) => void;
  disconnected: (reason?: string) => void;
  audioReceived: (audio: AudioChunk, participantId: string) => void;
  participantJoined: (identity: string, kind: string) => void;
  participantLeft: (identity: string, reason?: string) => void;
  error: (error: Error) => void;
}

/** LiveKit transport interface — pure audio transport only. */
export interface LiveKitTransport {
  /** Connect to a LiveKit room */
  connect(config: LiveKitConfig, roomName: string, token: string): Promise<void>;

  /** Disconnect from the current room */
  disconnect(): Promise<void>;

  /** Publish audio to the room */
  publishAudio(audio: AudioChunk): Promise<void>;

  /**
   * Subscribe to a specific transport event with type-safe callback.
   * @example transport.on('audioReceived', (audio) => { ... })
   */
  on<K extends keyof TransportEventMap>(event: K, handler: TransportEventMap[K]): void;

  /** Remove event handler */
  off<K extends keyof TransportEventMap>(event: K, handler: TransportEventMap[K]): void;

  /** Whether currently connected */
  readonly connected: boolean;
}

// ----- Provider Factory -----

/** Supported STT provider names. */
export type STTProviderName = 'whisper' | 'deepgram' | 'assemblyai';

/** Supported TTS provider names. */
export type TTSProviderName = 'elevenlabs' | 'openai' | 'piper' | 'kokoro';

/** Supported realtime provider names. */
export type RealtimeProviderName = 'openai-realtime';

/**
 * Factory for creating speech providers based on config.
 * Uses typed union keys to prevent typos — only known provider names compile.
 */
export interface ProviderFactory {
  /** Create an STT provider by name */
  createSTT(name: STTProviderName): STTProvider;

  /** Create a TTS provider by name */
  createTTS(name: TTSProviderName): TTSProvider;

  /** Create a realtime provider by name */
  createRealtime(name: RealtimeProviderName): RealtimeProvider;
}

// ----- Brain (Interview Intelligence) -----

/** Interview type — determines which prompting strategy the brain uses. */
export type InterviewType = 'behavioral' | 'technical' | 'system-design' | 'culture-fit' | 'mixed';

/** Context about the interview — resume, job, application, company. */
export interface InterviewContext {
  /** Candidate's resume content */
  resume: string;
  /** Job description */
  jobDescription: string;
  /** Application details */
  application?: {
    coverLetter?: string;
    status: string;
    appliedAt?: string;
  };
  /** Company research notes */
  companyResearch?: {
    name: string;
    industry?: string;
    notes: string;
  };
  /** Interview type — determines prompting strategy */
  interviewType?: InterviewType;
}

/** Brain's response to a transcript segment. */
export interface BrainResponse {
  /** What the agent should say (if anything) */
  speech?: string;
  /** Confidence in this response (0-1) */
  confidence: number;
  /** Strategy used */
  strategy: 'answer' | 'clarify' | 'defer' | 'silent';
  /** Light metadata for logging/debugging — kept small, not the full reasoning chain */
  metadata?: {
    /** Which context sources informed this response */
    sources?: string[];
    /** Topic being addressed */
    topic?: string;
    /** Time taken to generate response */
    responseTimeMs?: number;
  };
}

// ----- Context Retrieval -----

/**
 * Retrieves context from the backend for interview questions.
 *
 * Architecture:
 *   1. initialize(applicationId) — fetches all context ONCE at session init
 *   2. retrieve(query) — scores cached content in-memory (zero HTTP)
 *
 * Interview data never changes mid-interview — fetch once, cache forever.
 */
export interface ContextRetriever {
  /**
   * Fetch all context sources ONCE at session init.
   * Subsequent calls are no-ops (already cached).
   */
  initialize(applicationId: string): Promise<void>;

  /**
   * Retrieve relevant context for a question.
   * Runs on cached content — sub-millisecond, zero HTTP.
   * Returns ranked context chunks (most relevant first).
   *
   * @param query - The question or topic to retrieve context for
   * @param maxChunks - Maximum context chunks to return (default 5)
   */
  retrieve(query: string, maxChunks?: number): ContextChunk[];
}

/** A chunk of context retrieved for the brain. */
export interface ContextChunk {
  /** Source type: resume, job description, application, company notes */
  source: 'resume' | 'job' | 'application' | 'company';
  /** The context text */
  content: string;
  /** Relevance score (0-1), higher = more relevant */
  relevance: number;
  /** Optional metadata about the source */
  metadata?: {
    /** e.g., "React", "system design", "culture fit" */
    topic?: string;
    /** When this context was last updated */
    updatedAt?: string;
  };
}

/** Interview brain interface — where Ollama/OpenAI/Gemini plug in. */
export interface InterviewBrain {
  /** Initialize with interview context (resume, job, etc.) */
  initialize(context: InterviewContext): Promise<void>;

  /** Process a transcript segment and produce a response */
  process(segment: TranscriptSegment, memory: InterviewMemoryManager): Promise<BrainResponse>;

  /** Clean up resources */
  destroy(): Promise<void>;
}

/**
 * Interview memory — conversation history and extracted facts.
 *
 * ⚠️ recentTranscript is a ROLLING WINDOW, not full history.
 * Keep last N segments (default 50). Older segments are summarized into `summary`.
 * Prevents memory exhaustion and token explosion during long interviews.
 */
export interface InterviewMemoryState {
  /** Recent transcript segments (rolling window, max MAX_RECENT_SEGMENTS) */
  recentTranscript: TranscriptSegment[];
  /** Rolling summary of everything before the recent window */
  summary: string;
  /** Key facts extracted during the interview */
  facts: string[];
  /** Topics already covered */
  coveredTopics: string[];
  /** Questions the interviewer asked */
  questionsAsked: string[];
}

/**
 * Interview memory manager — pure behavior over InterviewMemoryState.
 *
 * ✅ NO LLM calls, NO async, NO prompt crafting.
 * ✅ Pure data transformers: add, trim, format.
 * ✅ Brain orchestrates summarization; manager applies results.
 *
 * Race condition prevention:
 *   - getSegmentsToSummarize() returns a snapshot
 *   - applySummary() uses the snapshot length, not current array state
 *   - On failure, data is preserved for retry
 *
 * Summary compaction:
 *   - Summary grows via append on each applySummary()
   -   needsCompaction() returns true when summary exceeds MAX_SUMMARY_LENGTH
   -   getSummaryForCompaction() returns the full summary for brain to re-summarize
   -   applyCompaction() replaces summary with compacted version
 */
export interface InterviewMemoryManager {
  /** Current state — read-only for brain, mutated only via manager methods */
  readonly state: InterviewMemoryState;

  /** Add a segment to the rolling window */
  addSegment(segment: TranscriptSegment): void;

  /** Get segments that need summarization (snapshot for safe async use) */
  getSegmentsToSummarize(): TranscriptSegment[];

  /** Apply a summary result and trim summarized segments by snapshot count */
  applySummary(newSummary: string, segmentCount: number): void;

  /** Check if summary exceeds length threshold and needs compaction */
  needsCompaction(): boolean;

  /** Get the full summary text for the brain to re-summarize */
  getSummaryForCompaction(): string;

  /** Replace summary with a compacted version (brain calls after re-summarizing) */
  applyCompaction(compactedSummary: string): void;

  /** Extract and store a fact from the conversation */
  addFact(fact: string): void;

  /** Track a topic covered in the interview */
  addCoveredTopic(topic: string): void;

  /** Track a question the interviewer asked */
  addQuestionAsked(question: string): void;

  /** Get memory state formatted for LLM prompt injection */
  toPromptContext(): string;
}

/** Maximum recent transcript segments to keep in memory before summarizing. */
export const MAX_RECENT_SEGMENTS = 50;

/** Maximum items per accumulated array before FIFO eviction. */
export const MAX_ACCUMULATED_ITEMS = 100;

/** Maximum summary character length before compaction is triggered. */
export const MAX_SUMMARY_LENGTH = 3000;

// ----- Brain Configuration (moved from hardcoded values) -----

/** Memory configuration */
export interface MemoryConfig {
  /** Maximum recent transcript segments to keep in memory before summarizing */
  max_recent_segments?: number;
  /** Segments to keep after summarization (rolling window tail) */
  keep_after_summarize?: number;
  /** Maximum items per accumulated array before FIFO eviction */
  max_accumulated_items?: number;
  /** Maximum summary character length before compaction is triggered */
  max_summary_length?: number;
}

/** Retriever configuration */
export interface RetrieverConfig {
  /** Request timeout for backend API calls (ms) */
  request_timeout_ms?: number;
  /** Maximum retries for transient failures (5xx, network errors) */
  max_retries?: number;
  /** Maximum content length for incoming content blocks */
  max_content_length?: number;
}

/** Prompt budget allocation (chars) */
export interface PromptBudget {
  system?: number;
  retrieval?: number;
  summary?: number;
  transcript?: number;
  question?: number;
}

/** LLM configuration */
export interface LLMConfig {
  /** Request timeout (ms) */
  timeout_ms?: number;
  /** Maximum retries for transient failures */
  max_retries?: number;
}

/** Responder configuration (extends LLM config with prompt budgets) */
export interface ResponderConfig {
  /** Ollama base URL */
  ollama_url?: string;
  /** Model name (e.g., "llama3.1", "mistral") */
  model?: string;
  /** LLM runtime settings */
  llm?: LLMConfig;
  /** Prompt budget allocation */
  prompt_budget?: PromptBudget;
  /** Memory ceilings for prompt budgeting */
  memory_ceilings?: {
    summary?: number;
    recent_transcript?: number;
    facts?: number;
    covered_topics?: number;
    questions_asked?: number;
  };
  /** Minimum speech length to consider raw output salvageable */
  min_salvageable_length?: number;
}

/** Planner configuration */
export interface PlannerConfig {
  /** Minimum transcript length to consider answering */
  min_substantive_length?: number;
  /** Maximum filler ratio before classifying as silent */
  max_filler_ratio?: number;
  /** Minimum content words to be considered a real question */
  min_content_words?: number;
  /** Keyword overlap threshold for duplicate question detection (0-1) */
  duplicate_threshold?: number;
}

/** Complete brain configuration — passed by session layer */
export interface BrainConfig {
  memory?: MemoryConfig;
  retriever?: RetrieverConfig;
  responder?: ResponderConfig;
  planner?: PlannerConfig;
}

// ----- Session Configuration -----

/** Configuration for an interview session. */
export interface VoiceSessionConfig {
  /** Interview mode */
  mode: InterviewMode;
  /** LiveKit room name */
  roomName: string;
  /** LiveKit token for joining */
  token: string;
  /** Application ID (for fetching context from backend) */
  applicationId: string;
  /** Provider configuration — typed to prevent typos */
  providers: {
    /** STT provider (used when mode is separate STT+TTS) */
    stt?: STTProviderName;
    /** TTS provider (used when mode is separate STT+TTS) */
    tts?: TTSProviderName;
    /** Realtime provider (used when provider natively combines STT+TTS) */
    realtime?: RealtimeProviderName;
  };
  /** Interview context (resume, job, etc.) — fetched from backend if not provided */
  context?: InterviewContext;
}

// ----- Session Events -----

/** Events emitted by an interview session. */
export type SessionEvent =
  | { type: 'started'; mode: InterviewMode; roomName: string }
  | { type: 'ended'; reason: string; transcript: TranscriptSegment[] }
  | { type: 'transcript'; segment: TranscriptSegment }
  | { type: 'agentSpeech'; text: string; confidence: number }
  | { type: 'suggestion'; text: string; reasoning: string }
  | { type: 'stateChanged'; state: SessionState }
  | { type: 'error'; error: Error; phase: 'connecting' | 'listening' | 'thinking' | 'speaking' | 'cleanup' };

/** Session event handler map — type-safe per event. */
export interface SessionEventMap {
  started: (mode: InterviewMode, roomName: string) => void;
  ended: (reason: string, transcript: TranscriptSegment[]) => void;
  transcript: (segment: TranscriptSegment) => void;
  agentSpeech: (text: string, confidence: number) => void;
  suggestion: (text: string, reasoning: string) => void;
  stateChanged: (state: SessionState) => void;
  error: (error: Error, phase: 'connecting' | 'listening' | 'thinking' | 'speaking' | 'cleanup') => void;
}

/**
 * Interview session orchestrator — the top-level engine.
 * Wires together: Transport (LiveKit) → VAD → Speech (STT/TTS/Realtime) → Brain (planner/responder).
 *
 * Lifecycle: start() → [listening → thinking → speaking loop] → stop()
 */
export interface InterviewSession {
  /** Current session state */
  readonly state: SessionState;

  /** Current interview mode */
  readonly mode: InterviewMode;

  /** Start the interview session — joins LiveKit room, initializes providers */
  start(config: VoiceSessionConfig): Promise<void>;

  /** Stop the interview session — disconnects, cleans up, returns final transcript */
  stop(reason: string): Promise<TranscriptSegment[]>;

  /** Force transition to a specific state (for error recovery, reconnects) */
  setState(state: SessionState, context?: { reason?: string; error?: Error }): void;

  /** Subscribe to session events with type-safe handlers */
  on<K extends keyof SessionEventMap>(event: K, handler: SessionEventMap[K]): void;

  /** Remove event handler */
  off<K extends keyof SessionEventMap>(event: K, handler: SessionEventMap[K]): void;
}
