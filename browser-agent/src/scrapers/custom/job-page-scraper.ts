import { BaseScraper, ScrapedJob } from '../base.js';
import { extractJsonLd } from './jsonld-extractor.js';
import { extractPageContent } from './content-extractor.js';
import { extractApplyUrl } from './apply-link-extractor.js';
import { resolveRedirect } from './redirect-resolver.js';
import { hashId, inferCompany, extractDomain, detectSourceFromUrl } from './helpers.js';
import { logger } from '../../utils/logger.js';

const log = logger.child({ component: 'JobPageScraper' });

const NAV_TIMEOUT_MS = 30_000;
const MAX_CONTENT_LENGTH = 20_000;
const MIN_CONTENT_LENGTH = 100;
const TASK_TIMEOUT_MS = 60_000;
const MAX_QUEUE_SIZE = 100;

/**
 * Scrape a single job page — tries JSON-LD first, then LLM.
 */
export async function scrapeSingleJobPage(
  scraper: BaseScraper,
  link: string,
  baseUrl: string
): Promise<ScrapedJob | null> {
  // Resolve listing URL against baseUrl
  let listingUrl: string;
  try {
    listingUrl = new URL(link, baseUrl).toString();
  } catch {
    listingUrl = baseUrl;
  }

  const page = await scraper.createPage();

  try {
    await page.goto(listingUrl, {
      waitUntil: 'domcontentloaded',
      timeout: NAV_TIMEOUT_MS,
    });

    // Try JSON-LD on individual page first
    const jsonLdJobs = await extractJsonLd(page, listingUrl);
    if (jsonLdJobs.length > 0) {
      log.debug({ link: listingUrl }, 'Job extracted via JSON-LD');
      return jsonLdJobs[0];
    }

    // Fallback: extract content and use LLM
    const content = await extractPageContent(page);
    if (!content || content.length < MIN_CONTENT_LENGTH) {
      log.debug({ link: listingUrl, contentLength: content?.length ?? 0 }, 'Page content too short');
      return null;
    }

    const extracted = await scraper.extractJobData(content.slice(0, MAX_CONTENT_LENGTH));
    if (!extracted) return null;

    // Extract apply link and resolve any redirects
    const applyUrl = await extractApplyUrl(page);
    const resolvedApplyUrl = applyUrl ? await resolveRedirect(applyUrl, scraper) : '';

    // Source should reflect where the job was LISTED, not where it applies
    const source = detectSourceFromUrl(listingUrl);

    return {
      external_id: hashId(listingUrl, extracted.title ?? '', extracted.company ?? ''),
      title: extracted.title ?? 'Unknown Title',
      company: extracted.company ?? inferCompany(baseUrl),
      location: extracted.location ?? 'Unknown',
      remote_type: extracted.remote_type ?? 'unknown',
      salary_min: extracted.salary_min ?? 0,
      salary_max: extracted.salary_max ?? 0,
      salary_currency: extracted.salary_currency ?? 'USD',
      description: extracted.description ?? '',
      requirements: extracted.requirements ?? '',
      url: listingUrl,
      application_url: resolvedApplyUrl || applyUrl || listingUrl,
      company_url: extractDomain(baseUrl),
      source,
    };
  } catch (err) {
    log.warn({ err, link: listingUrl }, 'Failed to scrape job page');
    return null;
  } finally {
    await page.close().catch(() => {});
  }
}

/**
 * Simple concurrency limiter — runs async functions with max parallelism.
 * Includes queue size limit and per-task timeout for backpressure.
 */
class ConcurrencyLimiter {
  private running = 0;
  private queue: Array<() => Promise<unknown>> = [];

  constructor(
    private readonly limit: number,
    private readonly taskTimeoutMs: number = TASK_TIMEOUT_MS,
    private readonly maxQueueSize: number = MAX_QUEUE_SIZE
  ) {}

  async run<T>(fn: () => Promise<T>): Promise<T> {
    if (this.queue.length >= this.maxQueueSize) {
      throw new Error(`Concurrency limiter queue full (max: ${this.maxQueueSize})`);
    }

    return new Promise<T>((resolve, reject) => {
      let timedOut = false;

      const timeoutId = setTimeout(() => {
        timedOut = true;
        reject(new Error(`Task timed out after ${this.taskTimeoutMs}ms`));
      }, this.taskTimeoutMs);

      this.queue.push(async () => {
        this.running++;
        try {
          const result = await fn();
          if (!timedOut) {
            clearTimeout(timeoutId);
            resolve(result);
          }
        } catch (err) {
          if (!timedOut) {
            clearTimeout(timeoutId);
            reject(err);
          }
        } finally {
          if (!timedOut) {
            this.running--;
            this.processQueue();
          }
        }
      });
      this.processQueue();
    });
  }

  private processQueue() {
    while (this.running < this.limit && this.queue.length > 0) {
      const task = this.queue.shift()!;
      task();
    }
  }
}

/**
 * Visit multiple job pages concurrently (with limit) and extract content.
 * @param scraper BaseScraper instance for creating pages and LLM extraction
 * @param jobLinks Array of job page URLs to scrape
 * @param baseUrl Base URL for resolving relative links
 * @param concurrencyLimit Maximum concurrent scraping tasks (default: 5)
 * @returns Array of successfully scraped jobs
 */
export async function scrapeJobPagesConcurrent(
  scraper: BaseScraper,
  jobLinks: string[],
  baseUrl: string,
  concurrencyLimit: number = 5
): Promise<ScrapedJob[]> {
  const limiter = new ConcurrencyLimiter(concurrencyLimit);
  const jobs: ScrapedJob[] = [];

  const results = await Promise.allSettled(
    jobLinks.map(link => limiter.run(() => scrapeSingleJobPage(scraper, link, baseUrl)))
  );

  for (const result of results) {
    if (result.status === 'fulfilled' && result.value) {
      jobs.push(result.value);
    }
  }

  return jobs;
}