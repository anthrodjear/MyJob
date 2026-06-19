/**
 * Brain Planner — Response strategy decision layer.
 *
 * Pure function: takes input + state → returns strategy plan.
 * No LLM calls, no async, no side effects.
 *
 * Strategy types:
 * - `answer` — generate a response (default for interview questions)
 * - `clarify` — ask for clarification (ambiguous or unclear input)
 * - `defer` — defer to human (off-topic or sensitive subject)
 * - `silent` — no response needed (small talk, filler, acknowledgment)
 */

import type { InterviewMemoryManager, TranscriptSegment, PlannerConfig } from '../types.js';
import { logger } from '../../utils/logger.js';

// ----- Types -----

export interface PlanResult {
  /** Response strategy to execute */
  strategy: 'answer' | 'clarify' | 'defer' | 'silent';
  /** Priority for processing (higher = more urgent) */
  priority: number;
  /** Reason for this decision (for logging/debugging) */
  reason: string;
  /** Whether responder should be invoked */
  shouldRespond: boolean;
  /** Additional metadata for responder */
  metadata?: {
    intent?: string;
    topic?: string;
    confidence?: number;
  };
}

// ----- Configuration -----

/** Thresholds for strategy decisions - defaults, can be overridden via PlannerConfig */
const DEFAULT_PLAN_CONFIG = {
  /** Minimum transcript length to consider answering */
  minSubstantiveLength: 5,
  /** Maximum filler ratio before classifying as silent */
  maxFillerRatio: 0.6,
  /** Minimum content words to be considered a real question */
  minContentWords: 2,
  /** Keyword overlap threshold for duplicate question detection (0-1) */
  duplicateThreshold: 0.5,
} as const;

// ----- Intent Patterns -----

/** Patterns that indicate the candidate is NOT asking a real question.
 *  Only match when the ENTIRE input is just an acknowledgment — no substantive content. */
const SILENT_PATTERNS: RegExp[] = [
  /^(yes|yeah|yep|yup|no|nope|nah|okay|ok|sure|alright|right|got it|i see|uh-huh|mhm|mm-hmm|ah|oh|wow|nice|cool|great|awesome|thank you|thanks|please|sorry|excuse me)[.!? ]*$/i,
  /^(hmm|hm|huh|umm|um|uh|er|ah|oh)[.!? ]*$/i,
  /^[.!?…]+$/,
];

/** Patterns that indicate a clarifying question is needed (explicit phrases only) */
const CLARIFY_PATTERNS: RegExp[] = [
  /\b(what do you mean|can you explain|how so|what do you mean by|i don't understand|could you clarify|sorry,? what|i'm not sure i follow|what does that mean)\b/i,
];

/** Patterns that indicate off-topic or sensitive subjects.
 *  Only match when the question is ABOUT the sensitive topic, not just mentioning it.
 *  "Can you tell me about remote work policy?" is legitimate — defer only when
 *  the topic IS the question, not referenced in passing. */
const DEFER_PATTERNS: RegExp[] = [
  /\b(salary|compensation|pay rate|how much (?:do you make|does this pay))\b/i,
  /\b(what (?:is|are) the (?:salary|compensation|benefits|vacation|pto|remote|onsite))\b/i,
  /\b(do you (?:offer|have|provide) (?:benefits|health insurance|vacation|pto|remote work))\b/i,
  /\b(politics|political|religion|religious)\b/i,
  /\b(confidential|trade secret|nda|non-disclosure|proprietary|classified)\b/i,
];

/** Small talk / acknowledgment words (for ratio calculation) */
const FILLER_WORDS = new Set([
  'yes', 'yeah', 'yep', 'yup', 'no', 'nope', 'nah',
  'okay', 'ok', 'sure', 'alright', 'right', 'got', 'it',
  'see', 'uh-huh', 'mhm', 'mm-hmm', 'ah', 'oh', 'wow',
  'nice', 'cool', 'great', 'awesome', 'thank', 'thanks',
  'please', 'sorry', 'excuse', 'hmm', 'hm', 'huh', 'umm',
  'um', 'uh', 'er',
]);

/** Common English stop words for content extraction */
const STOP_WORDS = new Set([
  'a', 'an', 'the', 'is', 'are', 'was', 'were', 'be', 'been', 'being',
  'have', 'has', 'had', 'do', 'does', 'did', 'will', 'would', 'could',
  'should', 'may', 'might', 'can', 'shall', 'to', 'of', 'in', 'for',
  'on', 'with', 'at', 'by', 'from', 'as', 'into', 'through', 'during',
  'before', 'after', 'above', 'below', 'between', 'under', 'again',
  'then', 'once', 'here', 'there', 'when', 'where', 'why', 'how',
  'all', 'both', 'each', 'few', 'more', 'most', 'other', 'some', 'such',
  'no', 'nor', 'not', 'only', 'own', 'same', 'so', 'than', 'too',
  'very', 'just', 'because', 'about', 'and', 'but', 'or', 'if', 'while',
  'i', 'you', 'he', 'she', 'it', 'we', 'they', 'me', 'him', 'her',
  'us', 'them', 'my', 'your', 'his', 'its', 'our', 'their', 'this',
  'that', 'these', 'those', 'what', 'which', 'who', 'whom',
]);

// ----- Core Planning Functions -----

/**
 * Extract content words (non-filler, non-stop words) from text.
 */
/**
 * Extract content words (non-filler, non-stop words) from text.
 * Preserves compound technical terms (C++, .NET, Node.js, React.js, etc.)
 * by treating dots, plus signs, and hash signs as part of words.
 */
function extractContentWords(text: string): string[] {
  // Preserve compound technical terms: C++, .NET, Node.js, React#, etc.
  return text
    .toLowerCase()
    .replace(/[^\w\s.+#-]/g, '')
    .split(/\s+/)
    .filter((word) => word.length > 0 && !STOP_WORDS.has(word.replace(/[.+#-]/g, '')) && !FILLER_WORDS.has(word.replace(/[.+#-]/g, '')));
}

/**
 * Calculate the ratio of filler words in text.
 */
function fillerRatio(text: string): number {
  const words = text.toLowerCase().split(/\s+/).filter((w) => w.length > 0);
  if (words.length === 0) return 1;
  const fillers = words.filter((w) => FILLER_WORDS.has(w.replace(/[^\w-]/g, '')));
  return fillers.length / words.length;
}

/**
 * Classify the intent of a transcript segment.
 */
function classifyIntent(segment: TranscriptSegment, config: typeof DEFAULT_PLAN_CONFIG): string {
  const text = segment.text.trim();

  // Check silent patterns first
  if (SILENT_PATTERNS.some((p) => p.test(text))) return 'acknowledgment';

  // Check clarify patterns
  if (CLARIFY_PATTERNS.some((p) => p.test(text))) return 'clarification-request';

  // Check defer patterns
  if (DEFER_PATTERNS.some((p) => p.test(text))) return 'off-topic';

  // Check if it's a question
  if (text.includes('?')) return 'question';

  // Check if it's substantive (has content words)
  const contentWords = extractContentWords(text);
  if (contentWords.length >= config.minContentWords) return 'substantive';

  return 'other';
}

/**
 * Calculate keyword overlap ratio between two content word sets.
 * Returns Jaccard-like similarity: |intersection| / |union|.
 */
function keywordOverlap(wordsA: string[], wordsB: string[]): number {
  if (wordsA.length === 0 || wordsB.length === 0) return 0;
  const setA = new Set(wordsA);
  const setB = new Set(wordsB);
  let intersection = 0;
  for (const word of setA) {
    if (setB.has(word)) intersection++;
  }
  const union = setA.size + setB.size - intersection;
  return union === 0 ? 0 : intersection / union;
}

/**
 * Check if current question is a duplicate of any previously asked question.
 * Uses keyword overlap (configurable threshold) instead of exact string match to catch rephrased questions.
 */
function isDuplicateQuestion(
  currentWords: string[],
  askedQuestions: string[],
  duplicateThreshold: number,
): boolean {
  if (currentWords.length === 0 || askedQuestions.length === 0) return false;

  for (const asked of askedQuestions) {
    const askedWords = extractContentWords(asked);
    const overlap = keywordOverlap(currentWords, askedWords);
    if (overlap >= duplicateThreshold) return true;
  }
  return false;
}

// ----- Main Planning Function -----

/**
 * Plan the response strategy for a transcript segment.
 *
 * Pure function — no side effects, no LLM calls, no async.
 * Returns a PlanResult indicating whether/how to respond.
 */
export function planResponse(
  segment: TranscriptSegment,
  memory: InterviewMemoryManager,
  _applicationId?: string,
  config?: PlannerConfig,
): PlanResult {
  const planConfig = {
    ...DEFAULT_PLAN_CONFIG,
    ...config,
  };
  const text = segment.text.trim();
  const intent = classifyIntent(segment, planConfig);

  // Filter out very short or empty input
  if (text.length < planConfig.minSubstantiveLength) {
    logger.debug({
      message: 'Plan: input too short, silent',
      length: text.length,
      ts: segment.timestamp,
    });
    return {
      strategy: 'silent',
      priority: 0,
      reason: 'Input too short to warrant a response',
      shouldRespond: false,
      metadata: { intent },
    };
  }

  // High filler ratio → likely acknowledgment, not a real question
  const ratio = fillerRatio(text);
  if (ratio >= planConfig.maxFillerRatio) {
    logger.debug({
      message: 'Plan: high filler ratio, silent',
      ratio: ratio.toFixed(2),
      ts: segment.timestamp,
    });
    return {
      strategy: 'silent',
      priority: 0,
      reason: `Filler ratio ${(ratio * 100).toFixed(0)}% exceeds threshold`,
      shouldRespond: false,
      metadata: { intent, confidence: 1 - ratio },
    };
  }

  // Off-topic / sensitive → defer to human
  if (intent === 'off-topic') {
    logger.info({
      message: 'Plan: off-topic detected, deferring',
      ts: segment.timestamp,
      text: text.substring(0, 50),
    });
    return {
      strategy: 'defer',
      priority: 3,
      reason: 'Off-topic or sensitive subject detected — defer to human operator',
      shouldRespond: true,
      metadata: { intent, topic: 'sensitive' },
    };
  }

  // Clarification request → ask for clarification
  if (intent === 'clarification-request') {
    logger.debug({
      message: 'Plan: clarification request detected',
      ts: segment.timestamp,
    });
    return {
      strategy: 'clarify',
      priority: 2,
      reason: 'Candidate requested clarification on previous response',
      shouldRespond: true,
      metadata: { intent },
    };
  }

  // Check memory for duplicate question (keyword overlap ≥50%)
  const memoryState = memory.state;
  const contentWords = extractContentWords(text);
  if (isDuplicateQuestion(contentWords, memoryState.questionsAsked, planConfig.duplicateThreshold)) {
    logger.debug({
      message: 'Plan: duplicate question detected, clarifying',
      ts: segment.timestamp,
    });
    return {
      strategy: 'clarify',
      priority: 1,
      reason: 'Duplicate question — ask candidate to elaborate',
      shouldRespond: true,
      metadata: { intent },
    };
  }

  // Check if topic was already covered — reference it in response
  if (
    contentWords.length > 0 &&
    memoryState.coveredTopics.some((topic) =>
      contentWords.some((w) => topic.toLowerCase().includes(w)),
    )
  ) {
    logger.debug({
      message: 'Plan: topic already covered, answering with reference',
      ts: segment.timestamp,
    });
    return {
      strategy: 'answer',
      priority: 1,
      reason: 'Topic previously covered — reference in response',
      shouldRespond: true,
      metadata: { intent, topic: 'previously-covered', confidence: 0.7 },
    };
  }

  // Default: answer the question
  const priority = intent === 'question' ? 2 : 1;

  logger.debug({
    message: 'Plan: will answer',
    strategy: 'answer',
    intent,
    ts: segment.timestamp,
  });

  return {
    strategy: 'answer',
    priority,
    reason: `Classified as ${intent} — generate response`,
    shouldRespond: true,
    metadata: { intent, confidence: 0.8 },
  };
}
