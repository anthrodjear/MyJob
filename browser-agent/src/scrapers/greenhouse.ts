import { ScrapedJob } from './base.js';
import { retry } from '../utils/retry.js';
import { logger } from '../utils/logger.js';

// ── Types ────────────────────────────────────────────────────────────

interface GreenhouseJob {
  id: number | string;
  title: string;
  company_name: string;
  location?: { name?: string };
  content?: string;
  absolute_url?: string;
  apply_url?: string;
  applyUrl?: string;
  salary_min?: number;
  salary_max?: number;
  salary_currency?: string;
  [key: string]: unknown;
}

interface GreenhouseBoardResponse {
  jobs?: GreenhouseJob[];
}

// ── Constants ────────────────────────────────────────────────────────

const API_TIMEOUT_MS = 30_000;
const PAGE_SIZE = 100;

/**
 * Scrape job listings from a Greenhouse job board.
 *
 * Uses the public Greenhouse JSON API (`boards-api.greenhouse.io`)
 * via `fetch()` — no browser needed. Fields are mapped directly from
 * structured JSON without LLM calls (Greenhouse already provides clean data).
 *
 * @param baseUrl - Greenhouse board URL (e.g. `https://boards.greenhouse.io/company`).
 * @param _keywords - Optional search keywords (not used by Greenhouse API).
 * @param _location - Optional location filter (not used by Greenhouse API).
 * @param signal - Optional AbortSignal for cancellation (combined with internal timeout).
 * @returns Array of normalized job postings.
 */
export async function scrapeGreenhouse(
  baseUrl: string,
  _keywords?: string[],
  _location?: string,
  signal?: AbortSignal
): Promise<ScrapedJob[]> {
  const scraper = new GreenhouseScraper();
  return scraper.scrape(baseUrl, signal);
}

class GreenhouseScraper {
  private log = logger.child({ component: 'GreenhouseScraper' });

  /**
   * Scrape all jobs from a Greenhouse board, handling pagination.
   */
  async scrape(baseUrl: string, signal?: AbortSignal): Promise<ScrapedJob[]> {
    const boardToken = this.extractBoardToken(baseUrl);
    this.log.info({ baseUrl, boardToken }, 'Scraping Greenhouse board');

    const allJobs: ScrapedJob[] = [];
    let start = 0;

    for (;;) {
      // Check for cancellation before each page
      signal?.throwIfAborted();

      const apiUrl =
        `https://boards-api.greenhouse.io/v1/boards/${boardToken}/jobs?content=true` +
        `&start=${start}&limit=${PAGE_SIZE}`;

      const pageJobs = await this.fetchPage(apiUrl, boardToken, signal);
      allJobs.push(...pageJobs);

      // Greenhouse returns fewer than PAGE_SIZE when there are no more results
      if (pageJobs.length < PAGE_SIZE) break;
      start += PAGE_SIZE;
    }

    this.log.info({ total: allJobs.length }, 'Greenhouse scrape complete');
    return allJobs;
  }

  /**
   * Fetch a single page of Greenhouse jobs with retry logic.
   * Returns an empty array on API errors (after logging).
   * Combines external AbortSignal with internal timeout.
   */
  private async fetchPage(apiUrl: string, boardToken: string, externalSignal?: AbortSignal): Promise<ScrapedJob[]> {
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
            throw new Error(`Greenhouse API HTTP ${response.status}`);
          }

          const contentType = response.headers.get('content-type') ?? '';
          if (!contentType.includes('application/json')) {
            throw new Error(`Greenhouse API returned non-JSON: ${contentType}`);
          }

          const data = await response.json() as GreenhouseBoardResponse;

          if (!data || !Array.isArray(data.jobs)) {
            throw new Error('Greenhouse API returned invalid response: missing jobs array');
          }

          // Map directly from structured JSON — no LLM needed for core fields
          const jobs: ScrapedJob[] = [];

          for (const job of data.jobs) {
            try {
              jobs.push(this.mapJob(job, boardToken));
            } catch (err) {
              this.log.warn({ err, jobId: job.id }, 'Failed to map Greenhouse job');
            }
          }

          return jobs;
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
          this.log.warn({ err, attempt, boardToken }, 'Retrying Greenhouse fetch'),
      },
    ).catch(err => {
      this.log.error({ err, boardToken }, 'Greenhouse fetch failed after retries');
      return [];
    });
  }

  /**
   * Map a raw Greenhouse job object to a ScrapedJob.
   * Deterministic — no LLM for core fields.
   */
  private mapJob(job: GreenhouseJob, boardToken: string): ScrapedJob {
    const jobId = String(job.id ?? 'unknown');
    const location = job.location ?? {};
    const locationName = String(location.name ?? 'Unknown');
    const description = String(job.content ?? '');

    // Prefer apply_url if provided by Greenhouse, otherwise use listing URL
    // Note: Greenhouse boards may redirect to external ATS (Workday, etc.)
    // The apply_url field is present on some boards but not all.
    const listingUrl = String(job.absolute_url ?? '');
    const applyUrl = String(job.apply_url ?? job.applyUrl ?? listingUrl);

    return {
      external_id: `greenhouse-${boardToken}-${jobId}`,
      title: String(job.title ?? ''),
      company: String(job.company_name ?? ''),
      location: locationName,
      remote_type: this.inferRemoteType(locationName),
      salary_min: job.salary_min ?? 0,
      salary_max: job.salary_max ?? 0,
      salary_currency: String(job.salary_currency ?? 'USD'),
      description,
      // TODO: Extract a proper requirements subsection from description
      // (e.g. look for "Requirements" / "Qualifications" header). For now
      // downstream consumers re-parse from description as needed.
      requirements: '',
      url: listingUrl,
      // Use the resolved apply URL (may differ from listing URL)
      application_url: applyUrl,
      company_url: '',
      source: 'greenhouse',
    };
  }

  /**
   * Extract the board token from a Greenhouse URL.
   * Handles:
   *   https://boards.greenhouse.io/company
   *   https://boards.greenhouse.io/company/jobs
   *   https://boards.greenhouse.io/embed/job_board?for=company
   *   Custom domains (e.g., jobs.company.com) — extracts first path segment
   */
  private extractBoardToken(url: string): string {
    try {
      const urlObj = new URL(url);

      // Try query param first (embed URLs use ?for=company)
      const forParam = urlObj.searchParams.get('for');
      if (forParam) return forParam;

      // Handle standard greenhouse.io subdomains: boards.greenhouse.io/company
      if (urlObj.hostname.endsWith('.greenhouse.io')) {
        return urlObj.hostname.replace('.greenhouse.io', '');
      }

      // Handle custom domains: jobs.company.com, careers.company.com, etc.
      // The board token is typically the first path segment
      const segments = urlObj.pathname.split('/').filter(Boolean);
      if (segments.length > 0) {
        return segments[0];
      }

      throw new Error(`Cannot extract board token from Greenhouse URL: ${url}`);
    } catch (err) {
      throw new Error(`Invalid Greenhouse URL: ${url} (${err instanceof Error ? err.message : String(err)})`);
    }
  }

  private inferRemoteType(locationName: string): 'remote' | 'hybrid' | 'onsite' | 'unknown' {
    const lower = locationName.toLowerCase();
    if (lower.includes('remote')) return 'remote';
    if (lower.includes('hybrid')) return 'hybrid';
    return 'onsite';
  }
}
