import { Page } from 'playwright';
import { BaseScraper, ScrapedJob } from '../base.js';
import { detectATS } from './ats-detector.js';
import { extractJsonLd, JSONLD_MIN_THRESHOLD } from './jsonld-extractor.js';
import { discoverWithPagination } from './pagination-discovery.js';
import { scrapeJobPagesConcurrent } from './job-page-scraper.js';
import { extractPageContent, autoScroll } from './content-extractor.js';
import { deduplicate } from './deduplicator.js';
import { hashId, inferCompany, extractDomain, detectSourceFromUrl } from './helpers.js';
import { logger } from '../../utils/logger.js';

const log = logger.child({ component: 'CustomScraper' });

export const MAX_CONTENT_LENGTH = 20_000;
const MIN_LLM_CONTENT_LENGTH = 500;
const NAV_TIMEOUT_MS = 30_000;

/**
 * Fallback scraper for unknown or custom career sites.
 *
 * Hybrid discovery strategy (orchestration):
 *   0. ATS detection — redirect to dedicated scraper if known
 *   1. JSON-LD structured data (JobPosting schema) — cheapest
 *   2. Pagination-aware link discovery — crawl multiple pages
 *   3. Concurrent job page extraction — parallel with concurrency limit
 *   4. LLM fallback — most expensive, last resort
 *
 * All heavy lifting is delegated to focused modules in custom/.
 */
export class CustomScraper extends BaseScraper {
  /**
   * Scrape job listings from an arbitrary career site URL.
   *
   * @param baseUrl - The career site URL to scrape.
   * @param keywords - Optional search keywords (logged, not used for filtering).
   * @param location - Optional location filter (logged, not used for filtering).
   * @returns Extracted and normalised job postings.
   */
  async scrape(baseUrl: string, keywords?: string[], location?: string): Promise<ScrapedJob[]> {
    log.info({ baseUrl, keywords, location }, 'Starting CustomScraper');

    const page = await this.createPage();

    try {
      // ── Step 0: ATS detection ─────────────────────────────────────
      const ats = await detectATS(page, baseUrl);
      if (ats) {
        log.info({ baseUrl, scraper: ats.name }, 'Detected known ATS, redirecting');
        try {
          return await ats.scraper(baseUrl);
        } catch (err) {
          log.warn({ err, baseUrl, ats: ats.name }, 'Dedicated ATS scraper failed, falling back to hybrid strategy');
          // Fall through to hybrid strategy instead of returning empty
        }
      }

      await page.goto(baseUrl, {
        waitUntil: 'domcontentloaded',
        timeout: NAV_TIMEOUT_MS,
      });

      // Scroll to load lazy content (non-blocking)
      await autoScroll(page).catch(err => log.warn({ err }, 'Auto-scroll failed, continuing'));

      // ── Step 1: JSON-LD extraction (cheapest) ────────────────────
      const jsonLdJobs = await extractJsonLd(page, baseUrl);
      if (jsonLdJobs.length >= JSONLD_MIN_THRESHOLD) {
        log.info({ count: jsonLdJobs.length }, 'Extracted jobs from JSON-LD (above threshold)');
        return deduplicate(jsonLdJobs);
      }

      if (jsonLdJobs.length > 0) {
        log.info({ count: jsonLdJobs.length }, 'JSON-LD found below threshold, continuing discovery');
      }

      // ── Step 2: Pagination-aware link discovery ──────────────────
      const allJobLinks = await discoverWithPagination(page, baseUrl);
      if (allJobLinks.length > 0) {
        log.info({ count: allJobLinks.length }, 'Discovered job links across pages');
        const scraped = await scrapeJobPagesConcurrent(this, allJobLinks, baseUrl);
        return deduplicate([...jsonLdJobs, ...scraped]);
      }

      // ── Step 3: LLM fallback (most expensive) ────────────────────
      // Re-navigate to baseUrl — discoverWithPagination may have moved the page
      await page.goto(baseUrl, {
        waitUntil: 'domcontentloaded',
        timeout: NAV_TIMEOUT_MS,
      });

      log.info({ baseUrl }, 'No structured data or job links found, using LLM fallback');
      const fallback = await this.llmFallback(page, baseUrl, location);
      return deduplicate([...jsonLdJobs, ...fallback]);
    } catch (err) {
      log.error({ err, baseUrl }, 'CustomScraper failed');
      return [];
    } finally {
      await page.close().catch(e => log.debug({ err: e }, 'Page close failed'));
    }
  }

  /**
   * Full-page LLM extraction — last resort.
   * Attempts to extract multiple jobs by chunking page content.
   */
  private async llmFallback(page: Page, baseUrl: string, location?: string): Promise<ScrapedJob[]> {
    const content = await extractPageContent(page);
    if (!content || content.length < MIN_LLM_CONTENT_LENGTH) {
      log.warn({ baseUrl, contentLength: content.length }, 'Page content too small for LLM extraction');
      return [];
    }

    // Extract all jobs from the page content
    const extractedJobs = await this.extractMultipleJobs(content.slice(0, MAX_CONTENT_LENGTH));
    if (!extractedJobs || extractedJobs.length === 0) return [];

    const jobs: ScrapedJob[] = [];

    for (const extracted of extractedJobs) {
      // For LLM fallback, we don't have individual job detail pages to visit for apply URLs.
      // Use the listing URL as application URL (same as JSON-LD path for hosted ATS).
      const listingUrl = extracted.url ?? baseUrl;

      const job: ScrapedJob = {
        external_id: hashId(listingUrl, extracted.title ?? '', extracted.company ?? ''),
        title: extracted.title ?? 'Unknown Title',
        company: extracted.company ?? inferCompany(baseUrl),
        location: extracted.location ?? location ?? 'Unknown',
        remote_type: extracted.remote_type ?? 'unknown',
        salary_min: extracted.salary_min ?? 0,
        salary_max: extracted.salary_max ?? 0,
        salary_currency: extracted.salary_currency ?? 'USD',
        description: extracted.description ?? '',
        requirements: extracted.requirements ?? '',
        url: listingUrl,
        application_url: listingUrl, // LLM fallback: listing page IS the application page for hosted ATS
        company_url: extractDomain(baseUrl),
        source: detectSourceFromUrl(listingUrl), // Use listing URL for source detection
      };

      jobs.push(job);
    }

    log.info({ baseUrl, jobsFound: jobs.length }, 'LLM fallback extraction complete');
    return jobs;
  }

  /**
   * Extract multiple job postings from page content using LLM.
   * The prompt should return an array of job objects.
   */
  private async extractMultipleJobs(rawContent: string): Promise<Array<{
    title: string | null;
    company: string | null;
    location: string | null;
    remote_type: 'remote' | 'hybrid' | 'onsite' | 'unknown';
    salary_min: number | null;
    salary_max: number | null;
    salary_currency: string | null;
    requirements: string;
    description: string;
    posted_at: string | null;
    url: string | null;
  }>> {
    // Use the existing extractJobData but with a modified prompt that returns multiple jobs
    // For now, fall back to single-job extraction (existing behavior)
    // TODO: Implement multi-job extraction prompt in config
    const single = await this.extractJobData(rawContent);
    if (!single) return [];

    return [{
      title: single.title,
      company: single.company,
      location: single.location,
      remote_type: single.remote_type,
      salary_min: single.salary_min,
      salary_max: single.salary_max,
      salary_currency: single.salary_currency,
      requirements: single.requirements,
      description: single.description,
      posted_at: single.posted_at,
      url: single.url,
    }];
  }
}