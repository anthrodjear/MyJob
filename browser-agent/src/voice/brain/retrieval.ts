/**
 * ContextRetriever — fetches interview context from the backend API.
 *
 * Architecture:
 *   1. Session init: fetchAll() loads resume, job, application, company ONCE
 *   2. Brain query: retrieve() scores cached content in-memory (zero HTTP)
 *   3. Interview data never changes mid-interview — fetch once, cache forever
 *
 * The backend owns the data. The retriever owns the fetching and ranking.
 * No LLM calls here — pure HTTP + simple scoring.
 */

import { logger } from '../../utils/logger.js';
import type { ContextChunk, ContextRetriever, RetrieverConfig } from '../types.js';

/** Maximum character length for incoming content blocks (prevents heap spikes). */
const DEFAULT_MAX_CONTENT_LENGTH = 10000;

/**
 * Creates a ContextRetriever that fetches from the backend API.
 *
 * @param backendUrl - Backend API base URL (e.g., "http://localhost:8080")
 * @param config - Retriever configuration
 */
export function createContextRetriever(
  backendUrl: string,
  config: RetrieverConfig = {},
): ContextRetriever {
  const requestTimeoutMs = config.request_timeout_ms ?? 5000;
  const maxRetries = config.max_retries ?? 2;
  const maxContentLength = config.max_content_length ?? DEFAULT_MAX_CONTENT_LENGTH;

  /** Cached chunks — populated once by fetchAll(), queried by retrieve(). */
  let cachedChunks: ContextChunk[] = [];

  /**
   * Fetch all context sources ONCE at session init.
   * Call this before the first retrieve() — subsequent calls are no-ops.
   */
  async function initialize(applicationId: string): Promise<void> {
    if (cachedChunks.length > 0) return; // Already initialized

    const encodedId = encodeURIComponent(applicationId);

    const [resumeChunk, jobChunk, applicationChunk, companyChunk] = await Promise.allSettled([
      fetchResume(backendUrl, encodedId, requestTimeoutMs, maxRetries, maxContentLength),
      fetchJobDescription(backendUrl, encodedId, requestTimeoutMs, maxRetries, maxContentLength),
      fetchApplication(backendUrl, encodedId, requestTimeoutMs, maxRetries, maxContentLength),
      fetchCompanyResearch(backendUrl, encodedId, requestTimeoutMs, maxRetries, maxContentLength),
    ]);

    const chunks: ContextChunk[] = [];

    if (resumeChunk.status === 'fulfilled' && resumeChunk.value) {
      chunks.push(resumeChunk.value);
    }
    if (jobChunk.status === 'fulfilled' && jobChunk.value) {
      chunks.push(jobChunk.value);
    }
    if (applicationChunk.status === 'fulfilled' && applicationChunk.value) {
      chunks.push(applicationChunk.value);
    }
    if (companyChunk.status === 'fulfilled' && companyChunk.value) {
      chunks.push(companyChunk.value);
    }

    cachedChunks = chunks;

    logger.info({
      message: 'ContextRetriever initialized',
      applicationId,
      chunksLoaded: chunks.length,
    });
  }

  /**
   * Retrieve relevant context for a question.
   * Runs keyword-overlap scoring on CACHED content — zero HTTP, sub-millisecond.
   */
  function retrieve(
    query: string,
    maxChunks: number = 5,
  ): ContextChunk[] {
    const limit = Math.max(1, maxChunks);

    // Score cached chunks — no HTTP, pure in-memory scoring
    const scored = cachedChunks.map((chunk) => ({
      ...chunk,
      relevance: computeRelevance(query, chunk.content),
    }));

    scored.sort((a, b) => b.relevance - a.relevance);
    return scored.slice(0, limit);
  }

  function destroy(): void {
    cachedChunks = [];
    logger.debug({ message: 'ContextRetriever destroyed' });
  }

  return { initialize, retrieve, destroy };
}

// ----- Backend API calls -----

async function fetchResume(
  backendUrl: string,
  applicationId: string,
  requestTimeoutMs: number,
  maxRetries: number,
  maxContentLength: number,
): Promise<ContextChunk | null> {
  try {
    const response = await fetchApi(backendUrl, `/api/v1/applications/${applicationId}/resume`, requestTimeoutMs, maxRetries);
    if (!response) return null;
    return {
      source: 'resume',
      content: truncate(str(response.content) ?? str(response.text) ?? '', maxContentLength),
      relevance: 0,
      metadata: { updatedAt: str(response.updatedAt) },
    };
  } catch (err) {
    logger.warn({ message: 'Failed to fetch resume', applicationId, err });
    return null;
  }
}

async function fetchJobDescription(
  backendUrl: string,
  applicationId: string,
  requestTimeoutMs: number,
  maxRetries: number,
  maxContentLength: number,
): Promise<ContextChunk | null> {
  try {
    const response = await fetchApi(backendUrl, `/api/v1/applications/${applicationId}/job`, requestTimeoutMs, maxRetries);
    if (!response) return null;
    return {
      source: 'job',
      content: truncate(str(response.description) ?? str(response.content) ?? '', maxContentLength),
      relevance: 0,
      metadata: {
        topic: str(response.title),
        updatedAt: str(response.postedAt),
      },
    };
  } catch (err) {
    logger.warn({ message: 'Failed to fetch job description', applicationId, err });
    return null;
  }
}

async function fetchApplication(
  backendUrl: string,
  applicationId: string,
  requestTimeoutMs: number,
  maxRetries: number,
  maxContentLength: number,
): Promise<ContextChunk | null> {
  try {
    const response = await fetchApi(backendUrl, `/api/v1/applications/${applicationId}`, requestTimeoutMs, maxRetries);
    if (!response) return null;
    const content = [
      str(response.coverLetter) && `Cover Letter:\n${str(response.coverLetter)}`,
      str(response.notes) && `Notes:\n${str(response.notes)}`,
      str(response.status) && `Status: ${str(response.status)}`,
    ]
      .filter(Boolean)
      .join('\n\n');
    if (!content) return null;
    return {
      source: 'application',
      content: truncate(content, maxContentLength),
      relevance: 0,
      metadata: { updatedAt: str(response.appliedAt) },
    };
  } catch (err) {
    logger.warn({ message: 'Failed to fetch application', applicationId, err });
    return null;
  }
}

async function fetchCompanyResearch(
  backendUrl: string,
  applicationId: string,
  requestTimeoutMs: number,
  maxRetries: number,
  maxContentLength: number,
): Promise<ContextChunk | null> {
  try {
    const response = await fetchApi(backendUrl, `/api/v1/applications/${applicationId}/company`, requestTimeoutMs, maxRetries);
    if (!response) return null;
    const content = [
      str(response.notes),
      str(response.industry) && `Industry: ${str(response.industry)}`,
      str(response.size) && `Company size: ${str(response.size)}`,
    ]
      .filter(Boolean)
      .join('\n\n');
    if (!content) return null;
    return {
      source: 'company',
      content: truncate(content, maxContentLength),
      relevance: 0,
      metadata: {
        topic: str(response.name),
        updatedAt: str(response.updatedAt),
      },
    };
  } catch (err) {
    logger.warn({ message: 'Failed to fetch company research', applicationId, err });
    return null;
  }
}

// ----- Helpers -----

/** Truncate content to prevent heap spikes from oversized responses. */
function truncate(content: string, maxContentLength: number): string {
  if (content.length <= maxContentLength) return content;
  logger.warn({
    message: 'Content truncated to prevent heap spike',
    original: content.length,
    truncated: maxContentLength,
  });
  return content.slice(0, maxContentLength);
}

/** Safely extract a string value from an API response. Converts non-string primitives to strings. */
function str(val: unknown): string | undefined {
  if (val === undefined || val === null) return undefined;
  if (typeof val === 'string') return val;
  if (typeof val === 'number' || typeof val === 'boolean') return String(val);
  return undefined;
}

async function fetchApi(
  backendUrl: string,
  path: string,
  requestTimeoutMs: number,
  maxRetries: number,
): Promise<Record<string, unknown> | null> {
  const base = backendUrl.endsWith('/') ? backendUrl.slice(0, -1) : backendUrl;
  const url = `${base}${path}`;

  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), requestTimeoutMs);

    try {
      const response = await fetch(url, { signal: controller.signal });

      if (response.ok) {
        const data: unknown = await response.json();
        if (data && typeof data === 'object' && !Array.isArray(data)) {
          return data as Record<string, unknown>;
        }
        return null;
      }

      if (response.status === 404) return null;

      if (response.status >= 500 && attempt < maxRetries) {
        logger.warn({
          message: 'Backend API transient error, retrying',
          status: response.status,
          attempt: attempt + 1,
          url,
        });
        await delay(500 * (attempt + 1));
        continue;
      }

      throw new Error(`Backend API error: ${response.status} ${response.statusText}`);
    } catch (err) {
      if (attempt < maxRetries && isTransientError(err)) {
        logger.warn({
          message: 'Backend API network error, retrying',
          err,
          attempt: attempt + 1,
          url,
        });
        await delay(500 * (attempt + 1));
        continue;
      }
      throw err;
    } finally {
      clearTimeout(timer);
    }
  }

  return null;
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function isTransientError(err: unknown): boolean {
  if (err instanceof Error) {
    return err.name === 'AbortError' || err.message.includes('fetch');
  }
  return false;
}

// ----- Relevance Scoring -----

/** Common English stop words — filtered out during tokenization. */
const STOP_WORDS = new Set([
  'the', 'and', 'for', 'are', 'but', 'not', 'you', 'all', 'can', 'had',
  'her', 'was', 'one', 'our', 'out', 'has', 'his', 'how', 'its', 'may',
  'new', 'now', 'old', 'see', 'way', 'who', 'did', 'get', 'let', 'say',
  'she', 'too', 'use', 'with', 'that', 'this', 'will', 'each', 'make',
  'like', 'long', 'look', 'many', 'some', 'than', 'them', 'then',
  'these', 'from', 'have', 'been', 'said', 'more', 'when', 'what',
  'your', 'they', 'would', 'could', 'should', 'about', 'which',
  'their', 'there', 'being', 'those', 'other', 'into', 'just', 'also',
  'than', 'very', 'does', 'done', 'doing', 'under', 'here', 'where',
  'while', 'since', 'still', 'after', 'before', 'between', 'both',
  'because', 'even', 'most', 'only', 'over', 'such', 'through',
  'during', 'much', 'well', 'back', 'down', 'able', 'upon',
]);

/**
 * Keyword-overlap relevance scoring with stop word filtering.
 * Fast, deterministic, good enough for ranking.
 */
function computeRelevance(query: string, content: string): number {
  if (!query || !content) return 0;
  const queryTerms = tokenize(query);
  const contentTerms = tokenize(content);
  if (queryTerms.length === 0 || contentTerms.length === 0) return 0;

  const contentSet = new Set(contentTerms);
  let matches = 0;
  for (const term of queryTerms) {
    if (contentSet.has(term)) matches++;
  }
  return matches / queryTerms.length;
}

function tokenize(text: string): string[] {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9\s]/g, ' ')
    .split(/\s+/)
    .filter((t) => t.length > 2 && !STOP_WORDS.has(t));
}
