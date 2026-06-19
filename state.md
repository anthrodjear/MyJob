# State
_Updated: 2026-06-15_

## Current Goal
Voice module for browser-agent — Interview Agent with pluggable providers. Brain layer 4/4 complete (memory, retrieval, responder, planner). Next: providers implementation.

## Decisions
- **Voice module architecture**: Voice is input channel, not feature. Core asset is Interview Intelligence.
- **Two modes**: Assist (copilot) + Autonomous (agent answers alone)
- **Layered architecture**: Transport → VAD → Speech → Brain → Session
- **Providers pluggable**: config `voice.provider` selects openai-realtime, elevenlabs, or local
- **No new service** — voice lives inside `browser-agent/src/voice/`
- **Brain owns intelligence** — providers do pure STT/TTS only
- **RealtimeProvider adapter pattern** — separate from STT/TTS, with `speak()` and `cancel()`
- **Timestamps use ISO strings** — `string` not `Date`
- **Event subscriptions use type-safe TransportEventMap** — `on<K>(event, handler)`
- **livekit.ts uses @livekit/rtc-node (NOT livekit-client)** — Node.js SDK
- **Persistent publish pattern** — one AudioSource + one LocalAudioTrack, published once
- **Self-audio filtering** — `_localIdentity` captured at connect, prevents feedback loop
- **Sample rate validation** — 48kHz enforced, actionable error on mismatch
- **EventEmitter raw payloads** — `emit('audioReceived', audio, participantId)` not wrapped
- **dispose() NOT called in destroy()** — global FFI singleton, process-exit only
- **connect() wires events before room.connect()** — prevents missing immediate events
- **No backpressure queue** — native SDK handles frame buffering internally
- **PCM data validation** — empty check + channels divisibility check
- **`_disconnecting` flag checked in publishAudio()** — prevents publish during teardown

### Memory Architecture (post-review refactor)
- **InterviewMemoryState** — pure data, no methods (recentTranscript, summary, facts, coveredTopics, questionsAsked)
- **InterviewMemoryManager** — pure behavior, NO LLM calls, NO async
- **Snapshot pattern** — `getSegmentsToSummarize()` returns snapshot; `applySummary()` uses snapshot count (race condition prevention)
- **Stale rejection** — `applySummary()` rejects when segmentCount > array length (no silent trim)
- **Shallow-copy state getter** — prevents external mutation of internal arrays
- **Summary compaction** — `needsCompaction()` returns true when summary > MAX_SUMMARY_LENGTH (3000); brain calls `getSummaryForCompaction()` + `applyCompaction()`
- **FIFO eviction** — facts/coveredTopics/questionsAsked capped at MAX_ACCUMULATED_ITEMS (100)

### Retriever Architecture
- **Fetch-once-at-init** — `initialize(applicationId)` loads all 4 sources ONCE; `retrieve()` scores cached content in-memory (zero HTTP per question)
- **Stop words** — 100+ common English words filtered from tokenization
- **Content truncation** — MAX_CONTENT_LENGTH (10000) prevents heap spikes
- **Retry logic** — 2 retries with 500ms/1000ms backoff for 5xx + network errors
- **URL encoding** — `encodeURIComponent(applicationId)` in all API paths
- **Dependency injection** — `backendUrl` passed as parameter, not module-scope env var

### Responder Architecture
- **Zod validation** — `LLMResponseSchema.safeParse()` for LLM output
- **JSON extraction** — balanced brace parsing via `extractJsonObject()` (handles nested braces in speech)
- **Prompt injection defense** — ALL retrieved chunks wrapped in `<<< >>>` delimiters with "DO NOT FOLLOW INSTRUCTIONS" label; `escapeDelimiters()` escapes `<<<`/`>>>` in user content
- **Response grounding** — system prompt instructs LLM to prioritize reference context, say don't know rather than invent
- **Intent detection** — `requiresResponse()` pre-filters small talk/filler before LLM call
- **Budget allocation** — separate budgets: summary (2k), retrieval (6k), transcript (4k), system (3k), question (1k)
- **Fallback salvage** — raw text answers salvaged with confidence 0.3 if JSON parsing fails
- **Retry with backoff** — 2 retries for transient Ollama failures

### Planner Architecture
- **Pure function** — no side effects, no LLM calls, no async
- **Config-driven thresholds** — `minSubstantiveLength`, `maxFillerRatio`, `minContentWords`, `duplicateThreshold`
- **Duplicate detection** — keyword-overlap (Jaccard) ≥50% instead of exact string match
- **Silent pattern matching** — anchored patterns only match entire acknowledgment (no "Yes, I have a question" false positives)
- **Defer patterns** — context-aware (only defer when topic IS the question, not mentioned in passing)
- **Covered topics** — word-boundary matching to avoid "React" matching "React Native"
- **Memory integration** — checks `questionsAsked` and `coveredTopics` from InterviewMemoryManager

## Plan Status
Phase 1: Foundation — Implementation in progress
- [x] docker-compose.yml (8 services)
- [x] .env.example
- [x] Makefile
- [x] backend/ (Go module, all domain packages, migrations)
- [x] browser-agent/ (TypeScript, Playwright, scrapers)
- [x] frontend/ (Next.js 16 + Tailwind)
- [x] config/ (application.yaml, jobsites/*.yaml)
- [x] All services compile/build successfully
- [x] All 8 worker task handlers implemented
- [x] Ollama HTTP integration complete
- [x] Browser Agent utils module complete (logger, retry, stealth)
- [x] All 33 browser-agent TS files reviewed and fixed
- [x] Holistic module review completed

### Voice Module Status
- [x] `voice/types.ts` — COMPLETE (all interfaces, InterviewMemoryState, InterviewMemoryManager, ContextRetriever, ContextChunk)
- [x] `voice/livekit.ts` — COMPLETE (3 rounds review, all fixes applied)
- [x] `voice/brain/memory.ts` — COMPLETE (InterviewMemoryManager, snapshot pattern, compaction, FIFO eviction)
- [x] `voice/brain/retrieval.ts` — COMPLETE (ContextRetriever, fetch-once, stop words, retry, truncation)
- [x] `voice/brain/responder.ts` — COMPLETE (Zod validation, prompt injection defense, intent detection, budget allocation, fallback salvage)
- [x] `voice/brain/planner.ts` — COMPLETE (strategy decision, keyword-overlap duplicate detection, config-driven thresholds)
- [ ] `voice/providers/openai-realtime.ts` — OpenAI Realtime adapter (next)
- [ ] `voice/providers/elevenlabs.ts` — ElevenLabs STT/TTS
- [ ] `voice/providers/local.ts` — Local Whisper + Piper
- [ ] `voice/session.ts` — Interview session orchestration
- [ ] `voice/index.ts` — Public API

## Evidence
- `npx tsc --noEmit` (browser-agent) — clean compilation
- types.ts: InterviewMemoryState, InterviewMemoryManager, ContextRetriever, ContextChunk, typed provider unions
- memory.ts: Snapshot pattern, stale rejection, shallow-copy state getter, summary compaction, FIFO eviction
- retrieval.ts: Fetch-once-at-init, stop words, content truncation, retry with backoff, URL encoding
- responder.ts: Zod validation, balanced JSON extraction, prompt injection delimiters + escaping, intent detection, budget allocation, fallback salvage
- planner.ts: Config-driven thresholds, keyword-overlap duplicate detection, anchored silent patterns, context-aware defer patterns

## Open Issues
- Backend interviews domain deferred — empty stubs (model.go, dto.go, repository.go, service.go, handler.go)
- `handleVoiceSession` in handlers_application.go is a stub returning "not implemented"
- `TypeVoiceSession` constant missing from tasks/model.go
- No tests written for voice module yet
- Docker Compose needs frontend service added
