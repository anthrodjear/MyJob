import { ScrapedJob } from './base.js';
import { retry } from '../utils/retry.js';
import { logger } from '../utils/logger.js';

// ── Constants ──────────────────────────────────────────────────────

const API_TIMEOUT_MS = 30_000;
const API_URL = 'https://remoteok.com/api';
const KNOWN_REMOTEOK_HOSTNAMES = ['remoteok.com', 'www.remoteok.com'] as const;

/** Error class for scraping failures — distinguishes from empty results. */
export class ScrapingError extends Error {
  constructor(message: string, public readonly cause?: Error) {
    super(message);
    this.name = 'ScrapingError';
  }
}

/**
 * Scrape remote job listings from RemoteOK.
 *
 * Uses the public RemoteOK JSON API via `fetch()` — no browser needed.
 * Fields are mapped directly from structured JSON without LLM calls.
 *
 * @param baseUrl - RemoteOK base URL (e.g. `https://remoteok.com`). Must be a RemoteOK domain.
 * @param keywords - Search keywords (used for client-side filtering after fetch).
 * @param _location - Location filter (typically ignored for RemoteOK — all jobs are remote).
 * @param signal - Optional AbortSignal for cancellation (combined with internal timeout).
 * @returns Array of normalized job postings.
 * @throws {ScrapingError} if the base URL is invalid or API fails after retries.
 */
export async function scrapeRemoteOK(
  baseUrl: string,
  keywords: string[],
  _location?: string,
  signal?: AbortSignal
): Promise<ScrapedJob[]> {
  const scraper = new RemoteOKScraper();
  return scraper.scrape(baseUrl, keywords, signal);
}

interface RemoteOKJob {
  id?: number | string;
  position?: string;
  company?: string;
  location?: string;
  tags?: string[];
  salary?: string;
  url?: string;
  description?: string;
  company_logo?: string;
  [key: string]: unknown;
}

class RemoteOKScraper {
  private log = logger.child({ component: 'RemoteOKScraper' });

  /**
   * Scrape all jobs from RemoteOK API with retry logic.
   */
  async scrape(baseUrl: string, keywords: string[], signal?: AbortSignal): Promise<ScrapedJob[]> {
    this.validateBaseUrl(baseUrl);
    this.log.info({ baseUrl, keywords }, 'Scraping RemoteOK API');

    const rawData = await this.fetchWithRetry(signal);

    // First item is metadata — skip it
    const jobData = rawData.slice(1);

    // Map directly from structured JSON — no LLM needed
    let jobs = jobData.map(job => this.mapJob(job));

    // Deduplicate by external_id — handle missing id with hash fallback
    const seen = new Set<string>();
    jobs = jobs.filter(job => {
      if (seen.has(job.external_id)) return false;
      seen.add(job.external_id);
      return true;
    });

    // Client-side keyword filtering
    if (keywords.length > 0) {
      const lower = keywords.map(k => k.toLowerCase());
      jobs = jobs.filter(job =>
        lower.some(kw =>
          `${job.title} ${job.company} ${job.description} ${job.requirements}`.toLowerCase().includes(kw),
        ),
      );
    }

    this.log.info({ total: jobs.length }, 'RemoteOK scrape complete');
    return jobs;
  }

  /**
   * Validate base URL against known RemoteOK hostnames.
   */
  private validateBaseUrl(baseUrl: string): void {
    try {
      const urlObj = new URL(baseUrl);
      const isKnownHost = KNOWN_REMOTEOK_HOSTNAMES.some(h => 
        urlObj.hostname === h || urlObj.hostname.endsWith(`.${h}`)
      );
      if (!isKnownHost) {
        throw new Error(`Invalid RemoteOK URL: ${baseUrl} (not a known RemoteOK domain)`);
      }
    } catch {
      throw new Error(`Invalid RemoteOK URL: ${baseUrl}`);
    }
  }

  /**
   * Fetch RemoteOK API with retry. Throws ScrapingError on failure after retries.
   * Combines external AbortSignal with internal timeout.
   */
  private async fetchWithRetry(externalSignal?: AbortSignal): Promise<RemoteOKJob[]> {
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
          const response = await fetch(API_URL, { signal: controller.signal });

          if (!response.ok) {
            throw new Error(`RemoteOK API HTTP ${response.status}`);
          }

          const contentType = response.headers.get('content-type') ?? '';
          if (!contentType.includes('application/json')) {
            throw new Error(`RemoteOK API returned non-JSON: ${contentType}`);
          }

          const data: unknown = await response.json();

          if (!Array.isArray(data)) {
            throw new Error(`RemoteOK API returned non-array: ${typeof data}`);
          }

          // Validate each item has expected structure
          for (const item of data) {
            if (!item || typeof item !== 'object') {
              throw new Error('RemoteOK API returned invalid job item');
            }
          }

          return data as RemoteOKJob[];
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
          this.log.warn({ err, attempt }, 'Retrying RemoteOK fetch'),
      },
    ).catch(err => {
      this.log.error({ err }, 'RemoteOK fetch failed after retries');
      throw new ScrapingError('RemoteOK fetch failed after retries', err);
    });
  }

  /**
   * Map a raw RemoteOK job object to a ScrapedJob.
   * Deterministic — no LLM for core fields.
   * Tags are mapped to requirements.
   */
  private mapJob(job: RemoteOKJob): ScrapedJob {
    const id = String(job.id ?? '');
    // Use hash of title+company if no id available for better deduplication
    const externalId = id
      ? `remoteok-${id}`
      : `remoteok-${this.hashFallback(job.position ?? '', job.company ?? '')}`;

    const salary = String(job.salary ?? '');
    const parsed = this.parseSalary(salary);

    const listingUrl = String(job.url ?? '');

    return {
      external_id: externalId,
      title: String(job.position ?? ''),
      company: String(job.company ?? ''),
      location: String(job.location ?? 'Worldwide'),
      remote_type: 'remote',
      salary_min: parsed.min,
      salary_max: parsed.max,
      salary_currency: parsed.currency,
      description: String(job.description ?? ''),
      // Map tags to requirements (skills/technologies)
      requirements: Array.isArray(job.tags) ? job.tags.join(', ') : '',
      url: listingUrl,
      // RemoteOK apply URL is the listing URL (links to company site)
      application_url: listingUrl,
      company_url: '',
      source: 'remoteok',
    };
  }

  /**
   * Simple hash for fallback external_id when id is missing.
   */
  private hashFallback(title: string, company: string): string {
    const str = `${title}|${company}`;
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      hash = ((hash << 5) - hash) + str.charCodeAt(i);
      hash |= 0;
    }
    return Math.abs(hash).toString(36);
  }

  /**
   * Parse salary range from strings like:
   *   "$100k - $180k"
   *   "$120k+"
   *   "USD 100k–130k"
   *   "£80k - £120k"
   *   "$80,000-$120,000"
   *   "$120,000+" (no k suffix)
   *   "Negotiable" / "Competitive" / "Market rate"
   *
   * Returns { min, max, currency } in actual numbers.
   */
  private parseSalary(salary: string): { min: number; max: number; currency: string } {
    if (!salary || !salary.trim()) return { min: 0, max: 0, currency: 'USD' };

    // Handle non-numeric salary descriptions
    const nonNumeric = ['negotiable', 'competitive', 'market rate', 'upon request', 'tbd'];
    const lowerSalary = salary.toLowerCase().trim();
    if (nonNumeric.some(n => lowerSalary.includes(n))) {
      return { min: 0, max: 0, currency: 'USD' };
    }

    // Detect currency symbol (default USD)
    let currency = 'USD';
    if (salary.includes('£')) currency = 'GBP';
    else if (salary.includes('€')) currency = 'EUR';
    else if (salary.includes('¥')) currency = 'JPY';
    else if (salary.toUpperCase().includes('USD')) currency = 'USD';

    // Match range: "$100k - $180k", "$120k+", "$80,000-$120,000", "USD 100k–130k", "£100k - €180k"
    const rangeMatch = salary.match(
      /[£€¥$]?\s*(\d[\d,]*)[kK]?\s*[+\-–]\s*[£€¥$]?\s*(\d[\d,]*)[kK]?/,
    );
    if (rangeMatch) {
      const useK = /[kK]/.test(salary);
      const min = parseInt(rangeMatch[1].replace(/,/g, ''), 10) * (useK ? 1000 : 1);
      const max = parseInt(rangeMatch[2].replace(/,/g, ''), 10) * (useK ? 1000 : 1);
      return { min, max, currency };
    }

    // Single value with + suffix: "$120k+", "$120,000+"
    const singlePlus = salary.match(/[£€¥$]?\s*(\d[\d,]*)[kK]?\+/);
    if (singlePlus) {
      const useK = /[kK]/.test(salary);
      const num = parseInt(singlePlus[1].replace(/,/g, ''), 10) * (useK ? 1000 : 1);
      return { min: num, max: num, currency };
    }

    // Single value without k: "$120,000" or "120000"
    const single = salary.match(/[£€¥$]?\s*(\d[\d,]*)/);
    if (single) {
      const useK = /[kK]/.test(salary);
      const num = parseInt(single[1].replace(/,/g, ''), 10) * (useK ? 1000 : 1);
      return { min: num, max: num, currency };
    }

    return { min: 0, max: 0, currency };
  }
}