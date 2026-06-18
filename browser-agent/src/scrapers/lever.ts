import { ScrapedJob } from './base.js';
import { retry } from '../utils/retry.js';
import { logger } from '../utils/logger.js';

// ── Constants ──────────────────────────────────────────────────────

const API_TIMEOUT_MS = 30_000;
const KNOWN_LEVER_HOSTNAMES = ['jobs.lever.co', 'api.lever.co'] as const;

/** Lever list item shape (requirements, responsibilities, etc.). */
interface LeverListItem {
  text: string;
  name?: string;
  content?: string;
}

/** Minimal Lever job shape from the API. */
interface LeverJob {
  id?: string | number;
  text?: string;
  company?: string;
  descriptionPlain?: string;
  hostedUrl?: string;
  companyUrl?: string;
  categories?: {
    location?: string;
    team?: string;
    department?: string;
    salaryCurrency?: string;
    [key: string]: unknown;
  };
  lists?: LeverListItem[];
  createdAt?: number;
  [key: string]: unknown;
}

/** Error class for scraping failures — distinguishes from empty results. */
export class ScrapingError extends Error {
  constructor(message: string, public readonly cause?: Error) {
    super(message);
    this.name = 'ScrapingError';
  }
}

/**
 * Scrape job listings from a Lever job board.
 *
 * Uses the public Lever JSON API (`api.lever.co`) via `fetch()` —
 * no browser needed. Fields are mapped directly from structured JSON.
 *
 * @param baseUrl - Lever board URL (e.g. `https://jobs.lever.co/company`).
 * @param _keywords - Optional search keywords (not used by Lever API).
 * @param _location - Optional location filter (not used by Lever API).
 * @param signal - Optional AbortSignal for cancellation (combined with internal timeout).
 * @returns Array of normalized job postings.
 * @throws {ScrapingError} if the board URL is invalid or API fails after retries.
 */
export async function scrapeLever(
  baseUrl: string,
  _keywords?: string[],
  _location?: string,
  signal?: AbortSignal
): Promise<ScrapedJob[]> {
  const scraper = new LeverScraper();
  return scraper.scrape(baseUrl, signal);
}

class LeverScraper {
  private log = logger.child({ component: 'LeverScraper' });

  /**
   * Scrape all jobs from a Lever board with retry logic.
   */
  async scrape(baseUrl: string, signal?: AbortSignal): Promise<ScrapedJob[]> {
    const boardName = this.extractBoardName(baseUrl);
    const log = this.log.child({ boardName });
    log.info({ baseUrl }, 'Scraping Lever board');

    const rawData = await this.fetchWithRetry(boardName, signal);

    if (!rawData) {
      throw new ScrapingError(`Failed to fetch Lever board after retries: ${boardName}`);
    }

    // Map directly from structured JSON — no LLM needed for core fields
    const jobs: ScrapedJob[] = [];
    const seen = new Set<string>();

    for (const job of rawData) {
      try {
        const mapped = this.mapJob(job, boardName);

        // Deduplicate by external_id
        if (seen.has(mapped.external_id)) continue;
        seen.add(mapped.external_id);

        jobs.push(mapped);
      } catch (err) {
        log.warn({ err, jobId: job.id }, 'Failed to map Lever job');
      }
    }

    log.info({ total: jobs.length }, 'Lever scrape complete');
    return jobs;
  }

  /**
   * Fetch Lever API with retry. Throws ScrapingError on failure after retries.
   * Combines external AbortSignal with internal timeout.
   */
  private async fetchWithRetry(boardName: string, externalSignal?: AbortSignal): Promise<LeverJob[]> {
    const apiUrl = `https://api.lever.co/v0/postings/${boardName}?mode=json`;

    return retry(
      async () => {
        const controller = new AbortController();
        const timer = setTimeout(() => controller.abort(), API_TIMEOUT_MS);

        // Combine external signal with internal timeout
        const onExternalAbort = () => controller.abort();
        if (externalSignal) {
          if (externalSignal.aborted) {
            controller.abort();
          } else {
            externalSignal.addEventListener('abort', onExternalAbort, { once: true });
          }
        }

        try {
          const response = await fetch(apiUrl, { signal: controller.signal });

          if (!response.ok) {
            throw new Error(`Lever API HTTP ${response.status}`);
          }

          const contentType = response.headers.get('content-type') ?? '';
          if (!contentType.includes('application/json')) {
            throw new Error(`Lever API returned non-JSON: ${contentType}`);
          }

          const data: unknown = await response.json();

          if (!Array.isArray(data)) {
            throw new Error(`Lever API returned non-array: ${typeof data}`);
          }

          return data as LeverJob[];
        } finally {
          clearTimeout(timer);
          if (externalSignal) {
            externalSignal.removeEventListener('abort', onExternalAbort);
          }
        }
      },
      {
        maxAttempts: 3,
        delay: 1000,
        onRetry: (err, attempt) =>
          this.log.warn({ err, attempt, boardName }, 'Retrying Lever fetch'),
      },
    ).catch(err => {
      this.log.error({ err, boardName }, 'Lever fetch failed after retries');
      throw new ScrapingError(`Lever fetch failed after retries: ${boardName}`, err);
    });
  }

  /**
   * Map a raw Lever job object to a ScrapedJob.
   * Deterministic — no LLM for core fields.
   */
  private mapJob(job: LeverJob, boardName: string): ScrapedJob {
    const jobId = String(job.id ?? 'unknown');
    const locationName = String(job.categories?.location ?? 'Unknown');

    // Extract requirements from structured lists with bullet prefix
    const requirements = Array.isArray(job.lists)
      ? job.lists
          .filter((l): l is LeverListItem => l != null && typeof l.text === 'string')
          .map(l => `- ${l.text}`)
          .join('\n')
      : '';

    const listingUrl = String(job.hostedUrl ?? '');
    const salaryCurrency = String(job.categories?.salaryCurrency ?? 'USD');

    return {
      external_id: `lever-${boardName}-${jobId}`,
      title: String(job.text ?? ''),
      company: String(job.company ?? 'Unknown'),
      location: locationName,
      remote_type: this.inferRemoteType(locationName),
      salary_min: 0,
      salary_max: 0,
      salary_currency: salaryCurrency,
      description: String(job.descriptionPlain ?? ''),
      requirements,
      url: listingUrl,
      // Lever apply URL is the hosted URL itself (apply form is on the same page)
      application_url: listingUrl,
      company_url: String(job.companyUrl ?? ''),
      source: 'lever',
    };
  }

  /**
   * Extract the board name from a Lever URL.
   * Throws on invalid URLs — never falls back to raw URL.
   *
   * Valid formats:
   *   https://jobs.lever.co/company
   *   https://jobs.lever.co/company/jobs
   *   https://jobs.lever.co/company/postings
   *   Custom domains (CNAME to jobs.company.com) — uses first path segment
   */
  private extractBoardName(url: string): string {
    const urlObj = new URL(url);

    // Validate against known Lever hostnames
    const isKnownHost = KNOWN_LEVER_HOSTNAMES.some(h => urlObj.hostname === h || urlObj.hostname.endsWith(`.${h}`));
    if (!isKnownHost) {
      // Check for custom domain (CNAME to Lever)
      // We accept any hostname but log a warning
      this.log.warn({ hostname: urlObj.hostname, url }, 'Custom Lever domain detected, extracting board name from path');
    }

    const segments = urlObj.pathname.split('/').filter(Boolean);

    if (!segments.length) {
      throw new Error(`Missing board name in Lever URL: ${url}`);
    }

    return segments[0];
  }

  private inferRemoteType(locationName: string): 'remote' | 'hybrid' | 'onsite' | 'unknown' {
    const lower = locationName.toLowerCase();
    if (lower.includes('remote')) return 'remote';
    if (lower.includes('hybrid')) return 'hybrid';
    if (lower === 'unknown' || lower === '') return 'unknown';
    return 'onsite';
  }
}