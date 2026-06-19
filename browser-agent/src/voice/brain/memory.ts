/**
 * InterviewMemoryManager — pure behavior over InterviewMemoryState.
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
 * Usage by brain:
 *   const segments = memory.getSegmentsToSummarize();
 *   const summaryText = await llmClient.generate(summarizePrompt);
 *   memory.applySummary(summaryText, segments.length);
 */

import { logger } from '../../utils/logger.js';
import type {
  InterviewMemoryManager,
  InterviewMemoryState,
  TranscriptSegment,
  MemoryConfig,
} from '../types.js';

/**
 * Creates an InterviewMemoryManager instance.
 *
 * @param config - Memory configuration (all values optional, defaults from config)
 */
export function createInterviewMemory(config: MemoryConfig = {}): InterviewMemoryManager {
  const maxSegments = config.max_recent_segments ?? 50;
  const keepAfterSummarize = config.keep_after_summarize ?? 10;
  const maxAccumulatedItems = config.max_accumulated_items ?? 100;
  const maxSummaryLength = config.max_summary_length ?? 3000;

  const state: InterviewMemoryState = {
    recentTranscript: [],
    summary: '',
    facts: [],
    coveredTopics: [],
    questionsAsked: [],
  };

  function addSegment(segment: TranscriptSegment): void {
    state.recentTranscript.push(segment);

    // Safety net: if window exceeds 2x max without summarization, trim to prevent
    // unbounded growth. Caller should invoke getSegmentsToSummarize() + applySummary()
    // before reaching this point.
    if (state.recentTranscript.length > maxSegments * 2) {
      const discarded = state.recentTranscript.length - maxSegments;
      logger.warn({
        message: 'InterviewMemory safety trim — window exceeded 2x max without summarization',
        current: state.recentTranscript.length,
        max: maxSegments,
        discarded,
      });
      state.recentTranscript.splice(0, discarded);
    }
  }

  function getSegmentsToSummarize(): TranscriptSegment[] {
    if (state.recentTranscript.length <= keepAfterSummarize) {
      return [];
    }
    // Return a snapshot — caller uses this count for applySummary()
    return state.recentTranscript.slice(0, -keepAfterSummarize);
  }

  function applySummary(newSummary: string, segmentCount: number): void {
    if (segmentCount <= 0) return;

    // Reject if snapshot count is stale — brain is out of sync with memory state.
    // Silently trimming would destroy incoming user responses.
    if (segmentCount > state.recentTranscript.length) {
      logger.error({
        message: 'applySummary rejected — segmentCount exceeds available segments (snapshot stale?)',
        requested: segmentCount,
        available: state.recentTranscript.length,
      });
      return;
    }

    // Apply summary — merge with existing if present
    state.summary = state.summary
      ? `${state.summary}\n${newSummary}`
      : newSummary;

    // Trim by snapshot count — safe from race conditions
    state.recentTranscript.splice(0, segmentCount);

    logger.info({
      message: 'InterviewMemory applied summary',
      segmentsTrimmed: segmentCount,
      remaining: state.recentTranscript.length,
      summaryLength: state.summary.length,
    });
  }

  function needsCompaction(): boolean {
    return state.summary.length > maxSummaryLength;
  }

  function getSummaryForCompaction(): string {
    return state.summary;
  }

  function applyCompaction(compactedSummary: string): void {
    if (!compactedSummary || !compactedSummary.trim()) return;
    const previousLength = state.summary.length;
    state.summary = compactedSummary.trim();
    logger.info({
      message: 'InterviewMemory applied compaction',
      previousLength,
      newLength: state.summary.length,
      reduction: previousLength - state.summary.length,
    });
  }

  function addFact(fact: string): void {
    if (!fact || !fact.trim()) return;
    if (state.facts.length >= maxAccumulatedItems) {
      state.facts.shift();
    }
    state.facts.push(fact.trim());
  }

  function addCoveredTopic(topic: string): void {
    if (!topic || !topic.trim()) return;
    if (state.coveredTopics.length >= maxAccumulatedItems) {
      state.coveredTopics.shift();
    }
    state.coveredTopics.push(topic.trim());
  }

  function addQuestionAsked(question: string): void {
    if (!question || !question.trim()) return;
    if (state.questionsAsked.length >= maxAccumulatedItems) {
      state.questionsAsked.shift();
    }
    state.questionsAsked.push(question.trim());
  }

  function toPromptContext(): string {
    const parts: string[] = [];

    if (state.summary) {
      parts.push(`## Interview Summary\n${state.summary}`);
    }

    if (state.facts.length > 0) {
      parts.push(`## Key Facts\n${state.facts.map((f) => `- ${f}`).join('\n')}`);
    }

    if (state.coveredTopics.length > 0) {
      parts.push(`## Topics Covered\n${state.coveredTopics.map((t) => `- ${t}`).join('\n')}`);
    }

    if (state.questionsAsked.length > 0) {
      parts.push(`## Questions Asked\n${state.questionsAsked.map((q) => `- ${q}`).join('\n')}`);
    }

    if (state.recentTranscript.length > 0) {
      const transcriptLines = state.recentTranscript
        .map((s) => {
          const confidenceStr = (typeof s.confidence === 'number' && !Number.isNaN(s.confidence))
            ? ` (${Math.round(s.confidence * 100)}%)`
            : '';
          return `${s.speaker}${confidenceStr} [${s.timestamp}]: ${s.text}`;
        })
        .join('\n');
      parts.push(`## Recent Transcript\n${transcriptLines}`);
    }

    return parts.join('\n\n');
  }

  return {
    get state() {
      // Return shallow copies — prevents external mutation of internal arrays
      return {
        recentTranscript: [...state.recentTranscript],
        summary: state.summary,
        facts: [...state.facts],
        coveredTopics: [...state.coveredTopics],
        questionsAsked: [...state.questionsAsked],
      };
    },
    addSegment,
    getSegmentsToSummarize,
    applySummary,
    needsCompaction,
    getSummaryForCompaction,
    applyCompaction,
    addFact,
    addCoveredTopic,
    addQuestionAsked,
    toPromptContext,
  };
}
