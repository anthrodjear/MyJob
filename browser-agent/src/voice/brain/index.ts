/**
 * Brain — owns all interview intelligence components.
 *
 * Composes: planner + responder + memory + retriever
 * Exposes ONE method to session: respond(segment)
 *
 * Session doesn't know about planning, retrieval, memory management,
 * or LLM generation — it just calls brain.respond() and gets a response.
 */

import { logger } from '../../utils/logger.js';
import { createInterviewMemory } from './memory.js';
import { createContextRetriever } from './retrieval.js';
import { createResponder } from './responder.js';
import { planResponse } from './planner.js';
import type {
  Brain,
  BrainConfig,
  BrainResponse,
  InterviewContext,
  InterviewMemoryManager,
  TranscriptSegment,
} from '../types.js';

const log = logger.child({ component: 'Brain' });

/**
 * Creates a Brain instance that owns all interview intelligence.
 *
 * @param backendUrl - Backend API URL for context retrieval
 * @param config - Brain configuration (memory, retriever, responder, planner)
 */
export function createBrain(
  backendUrl: string,
  config: BrainConfig = {},
): Brain {
  const memory: InterviewMemoryManager = createInterviewMemory(config.memory);
  const contextRetriever = createContextRetriever(backendUrl, config.retriever);
  const responder = createResponder(config.responder ?? {});

  let interviewContext: InterviewContext | null = null;
  let initialized = false;

  async function initialize(context: InterviewContext): Promise<void> {
    if (initialized) return;

    interviewContext = context;
    initialized = true;

    log.info({ message: 'Brain initialized' });
  }

  async function respond(segment: TranscriptSegment): Promise<BrainResponse> {
    if (!initialized) {
      log.warn({ message: 'Brain not initialized, returning fallback' });
      return {
        speech: 'Could you repeat the question?',
        confidence: 0,
        strategy: 'clarify',
        metadata: { sources: ['not-initialized'] },
      };
    }

    // 1. Add to memory
    memory.addSegment(segment);

    // 2. Plan response strategy
    const plan = planResponse(segment, memory);

    if (!plan.shouldRespond) {
      log.debug({ message: 'Plan: no response needed', reason: plan.reason });
      return {
        speech: undefined,
        confidence: 1,
        strategy: 'silent',
        metadata: { sources: ['planner'], topic: plan.metadata?.intent },
      };
    }

    // 3. Retrieve relevant context
    const contextChunks = contextRetriever.retrieve(segment.text);

    // 4. Generate response
    const response = await responder.generate(
      segment,
      memory,
      contextChunks,
      interviewContext ?? { resume: '', jobDescription: '' },
    );

    // 5. Track memory updates based on response
    if (response.speech && response.strategy === 'answer') {
      memory.addCoveredTopic(segment.text.slice(0, 100));
    }
    if (segment.speaker === 'interviewer' || segment.speaker === 'user') {
      memory.addQuestionAsked(segment.text);
    }

    return response;
  }

  function destroy(): void {
    responder.destroy();
    contextRetriever.destroy();
    initialized = false;
    log.info({ message: 'Brain destroyed' });
  }

  return {
    get memory() {
      return memory;
    },
    initialize,
    respond,
    destroy,
  };
}
