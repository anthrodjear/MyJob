# Session Log

## 2026-06-15 16:30 [saved]
Goal: Voice module types.ts and livekit.ts — Interview Agent for browser-agent
Decisions:
- Voice module uses @livekit/rtc-node (Node.js SDK) not livekit-client (browser SDK) — livekit.ts rewritten
- Persistent publish pattern: one AudioSource + one LocalAudioTrack, published once via initPublish()
- AudioStream receive uses EventEmitter (`frameReceived` event), not ReadableStream
- Self-audio filtering: capture `_localIdentity` at connect, skip own tracks to prevent infinite STT→Brain→TTS feedback loop
- Sample rate validation: 48kHz enforced, non-48kHz audio rejected with actionable error
- EventEmitter raw payloads: `emit('audioReceived', audio, participantId)` not wrapped `{type, audio}`
- No application-level backpressure queue — native SDK handles frame buffering internally
- PCM validation: empty frames skipped, length must be divisible by channel count
- connect() wires events BEFORE room.connect() to prevent missing immediate events
- dispose() NOT called in destroy() — global FFI singleton, process-exit only
- InterviewMemory uses rolling window (max 50 segments + summary) to prevent token explosion
- Brain owns intelligence, providers do pure STT/TTS only — two modes: Assist + Autonomous
Rejected: Browser SDK (`livekit-client`) for Node.js backend — AudioStream not available, AudioContext/browser APIs needed
Open: Voice brain layer (memory, retrieval, responder, planner), providers (openai-realtime, elevenlabs, local), session orchestration, index.ts. Backend interviews domain deferred.

## 2026-06-15 18:00 [saved]
Goal: Voice brain layer — memory.ts, retrieval.ts, responder.ts
Decisions:
- Split InterviewMemory into InterviewMemoryState (pure data) + InterviewMemoryManager (behavior, no LLM awareness)
- Snapshot pattern: getSegmentsToSummarize() returns snapshot, applySummary() uses snapshot count (race condition prevention)
- Stale rejection: applySummary() rejects when segmentCount > array length — no silent trim (data loss prevention)
- Shallow-copy state getter: prevents external mutation of internal arrays
- Summary compaction: needsCompaction() when summary > 3000 chars, brain calls getSummaryForCompaction() + applyCompaction()
- FIFO eviction: facts/coveredTopics/questionsAsked capped at 100 items
- ContextRetriever: fetch-once-at-init (initialize loads 4 sources), retrieve() scores cached content in-memory (zero HTTP per question)
- Stop words: 100+ common English words filtered from tokenization
- Content truncation: MAX_CONTENT_LENGTH (10000) prevents heap spikes from oversized responses
- Responder: Zod validation (LLMResponseSchema.safeParse), JSON extraction via indexOf/lastIndexOf (no greedy regex)
- Prompt injection defense: ALL retrieved chunks wrapped in <<< >>> delimiters with "DO NOT FOLLOW INSTRUCTIONS" label
- Response grounding: system prompt instructs LLM to prioritize reference context, say don't know rather than invent
- Intent detection: requiresResponse() pre-filters small talk/filler before LLM call (saves tokens)
- Budget allocation: separate budgets — summary (2k), retrieval (6k), transcript (4k), system (3k), question (1k)
- Fallback salvage: raw text answers salvaged with confidence 0.3 if JSON parsing fails
- Retry with backoff: 2 retries for transient Ollama failures, 500ms/1000ms delay
Rejected: Memory as single interface with summarize() accepting LLM callback — violates pure data principle, race conditions on async boundary
Open: planner.ts (next), providers (openai-realtime, elevenlabs, local), session orchestration, index.ts. Backend interviews domain deferred.

## 2026-06-15 19:30 [saved]
Goal: Voice brain layer — planner.ts completion + responder.ts blocker fixes
Decisions:
- Planner: Pure decision function with config-driven thresholds (minSubstantiveLength, maxFillerRatio, minContentWords, duplicateThreshold)
- Duplicate detection: keyword-overlap (Jaccard) ≥50% instead of exact string match — catches rephrased questions
- Silent patterns anchored: `^...$` prevents "Yes, I have a question" false positives
- Defer patterns context-aware: only defer when topic IS the question ("what is the salary"), not when mentioned in passing
- Covered topics word-boundary matching: "React" doesn't match "React Native" — uses extractContentWords() intersection
- Responder security: escapeDelimiters() added — escapes `<<<`→`<&lt;&lt;`, `>>>`→`&gt;&gt;&gt;` in ALL user content (resume, job, interviewer speech)
- Responder JSON extraction: extractJsonObject() with balanced brace parsing — handles nested braces in speech field (e.g., "I use {key: val}")
- All 4 brain files complete and compiling cleanly
Rejected: Exact string duplicate detection; greedy regex JSON extraction; unescaped delimiters in user content
Open: providers (openai-realtime, elevenlabs, local), session orchestration, index.ts. Backend interviews domain deferred.

## 2026-06-24 [saved]
Goal: System Config frontend implementation + code quality cleanup
Decisions:
- Removed all `// ---------------------------------------------------------------------------` separator lines from frontend files — user preference for clean comments
- Config types use PascalCase for scoring.Weights (Skill, Experience, etc.) — Go struct has no JSON tags
- SystemConfigResponse shape: `{ config: EffectiveConfig, version?: string }` matching backend DTO
- useSystemConfig hook returns `{ config, version }` — not just EffectiveConfig
- Each config section component is independent — saves via PATCH per field, no shared form state
- IntegrationsSection is read-only display — status determined by backend, not editable
- SystemConfigPageClient embedded in SettingsPageClient via "system" tab — not a separate route
- SettingsPageClient updated: 5 tabs (preferences, skills, education, links, system)
- All config section components follow same pattern: controlled state → compare → PATCH → invalidate
- Email & Interview combined into one section — both are operational settings
- Approval tiers use fieldset/legend for grouped form sections — proper ARIA semantics
Files created/modified:
- `frontend/src/lib/types/config.ts` — rewritten without separator lines
- `frontend/src/hooks/useSystemConfig.ts` — rewritten without separator lines, returns { config, version }
- `frontend/src/lib/api/config.ts` — unchanged (already clean)
- `frontend/src/components/config/ScoringSection.tsx` — removed separator lines
- `frontend/src/components/config/LLMSection.tsx` — NEW
- `frontend/src/components/config/VoiceSection.tsx` — NEW
- `frontend/src/components/config/ApprovalTiersSection.tsx` — NEW
- `frontend/src/components/config/AutomationSection.tsx` — NEW
- `frontend/src/components/config/EmailInterviewSection.tsx` — NEW
- `frontend/src/components/config/IntegrationsSection.tsx` — NEW
- `frontend/src/app/dashboard/settings/config/SystemConfigPageClient.tsx` — NEW
- `frontend/src/app/dashboard/settings/config/page.tsx` — NEW
- `frontend/src/app/dashboard/settings/SettingsPageClient.tsx` — updated (system tab + removed separators)
- `context/progress-tracker.md` — updated (systemconfig domain, API endpoint, dates)

## 2026-06-24 [saved]
Goal: Code review fixes for System Config frontend module
Decisions:
- BLOCKER B1: Sequential mutation counter pattern replaced with executeOverrides helper using mutateAsync + Promise.allSettled — partial failures now logged, onSaved always called
- BLOCKER B2: Undefined tokens fixed — bg-error/10 → bg-danger-light, text-error-dark → text-danger-dark, bg-success/10 → bg-success-light (matching globals.css design tokens)
- BLOCKER B3: bg-card → bg-surface (no --color-card in theme)
- WARN W1: Added bg-surface to all input/select elements via shared INPUT_CLASS constant
- WARN W2: VoiceSection LiveKit API key changed to type="password"
- WARN W4: SystemConfigPageClient h1 → h2 (nested inside SettingsPageClient h1)
- IntegrationsSection status styles fixed: bg-success/10 → bg-success-light, bg-error/10 → bg-danger-light
- SettingsPageClient: bg-card → bg-surface, bg-success/10 → bg-success-light
- executeOverrides exported from useSystemConfig.ts for reuse across all section components
- All 6 section components now use shared INPUT_CLASS for consistent input styling
- TypeScript compiles cleanly, 75 tests pass
