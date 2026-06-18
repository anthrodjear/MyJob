import { Locator, Page } from 'playwright';
import { BaseScraper, ScrapedJob } from './base.js';
import crypto from 'node:crypto';

// ── Constants ──────────────────────────────────────────────────────

const NAV_TIMEOUT_MS = 60_000;
const SELECTOR_TIMEOUT_MS = 6_000; // Per-selector timeout
const CLICK_TIMEOUT_MS = 5_000;
const DETAIL_TIMEOUT_MS = 5_000;
const MAX_JOBS_PER_PAGE = 20;
const MAX_PAGES = 5;
// Unit separator character for hash delimiter (non-printable, unlikely in titles)
const HASH_DELIMITER = '\x1f';

/** Fallback selectors — Indeed changes DOM attributes frequently. */
const JOB_CARD_SELECTORS = [
  '[data-testid="job-card"]',
  '.job_seen_beacon',
  '[data-jk]',
  '.resultContent',
  '.tapItem',
] as const;

/** Job description container selector — used to target only the description for LLM extraction. */
const JOB_DESCRIPTION_SELECTORS = [
  '[id="jobDescriptionText"]',
  '.jobsearch-JobComponent-description',
] as const;

/** Anti-bot challenge indicators. */
const ANTIBOT_PATTERNS = [
  'verify you are human',
  'access denied',
  'blocked',
  'unusual traffic',
  'cf-challenge-running',
  'data-ray',
] as const;

/**
 * Scrape job listings from Indeed.
 *
 * Indeed requires JavaScript rendering — this scraper extends BaseScraper
 * for Playwright browser lifecycle management.
 *
 * Strategy:
 *   1. Extract title, company, location from DOM (deterministic)
 *   2. Use LLM only for requirements, salary parsing, remote type inference
 *   3. Stable IDs via SHA-256 hash for deduplication
 *   4. Handle pagination via "Next" page link
 *
 * @param baseUrl - Indeed base URL (e.g. `https://www.indeed.com`).
 * @param keywords - Search keywords.
 * @param location - Location filter.
 * @param signal - Optional AbortSignal for cancellation (checked at key navigation points).
 * @returns Array of normalized job postings.
 */
export async function scrapeIndeed(
  baseUrl: string,
  keywords: string[],
  location: string,
  signal?: AbortSignal
): Promise<ScrapedJob[]> {
  const scraper = new IndeedScraper();
  try {
    await scraper.init();
    return await scraper.scrape(baseUrl, keywords, location, signal);
  } finally {
    await scraper.close();
  }
}

class IndeedScraper extends BaseScraper {
  async scrape(baseUrl: string, keywords: string[], location: string, signal?: AbortSignal): Promise<ScrapedJob[]> {
    const page = await this.newPage();
    const jobs: ScrapedJob[] = [];
    const seen = new Set<string>();

    try {
      // Build search URL
      const searchParams = new URLSearchParams({
        q: keywords.join(' '),
        l: location,
      });
      const initialUrl = `${baseUrl}/jobs?${searchParams.toString()}`;
      let currentUrl: string | null = initialUrl;

      for (let pageNum = 0; pageNum < MAX_PAGES && currentUrl; pageNum++) {
        // Check for cancellation before each page navigation
        signal?.throwIfAborted();

        await page.goto(currentUrl, { waitUntil: 'domcontentloaded', timeout: NAV_TIMEOUT_MS });

        // Anti-bot detection
        if (await this.isBlocked(page)) {
          this.log.warn({ url: page.url(), pageNum }, 'Indeed anti-bot challenge detected');
          break;
        }

        // Wait for job cards — try fallback selectors
        const foundSelector = await this.waitForJobCards(page);
        if (!foundSelector) {
          this.log.warn({ pageNum }, 'No job cards found on page');
          break;
        }

        // Get all job cards
        const jobCards = await this.findJobCards(page);

        for (const card of jobCards.slice(0, MAX_JOBS_PER_PAGE)) {
          // Check for cancellation before each job card
          signal?.throwIfAborted();

          try {
            const job = await this.extractJobFromCard(card, page);
            if (job && !seen.has(job.external_id)) {
              seen.add(job.external_id);
              jobs.push(job);
            }
          } catch (err) {
            this.log.warn({ err }, 'Failed to extract job from card');
          }
        }

        // Find next page URL
        currentUrl = await this.findNextPageUrl(page, currentUrl);
        if (currentUrl) {
          this.log.debug({ pageNum, nextUrl: currentUrl }, 'Following pagination');
        }
      }
    } finally {
      // page.context().close() is handled by BaseScraper.close()
    }

    return jobs;
  }

  // ── Anti-bot detection ──────────────────────────────────────────────

  /**
   * Check if the page is showing a CAPTCHA or access denied page.
   */
  private async isBlocked(page: Page): Promise<boolean> {
    const url = page.url();
    if (url.includes('captcha') || url.includes('challenge')) return true;

    // Lighter check: use page.content() only if URL check didn't match
    try {
      const content = await page.content();
      return ANTIBOT_PATTERNS.some(pattern => content.includes(pattern));
    } catch {
      return false;
    }
  }

  // ── Job card discovery ──────────────────────────────────────────────

  /**
   * Wait for job cards using fallback selectors.
   * Returns the selector that succeeded, or null if none matched.
   */
  private async waitForJobCards(page: Page): Promise<string | null> {
    for (const selector of JOB_CARD_SELECTORS) {
      try {
        await page.waitForSelector(selector, { timeout: SELECTOR_TIMEOUT_MS });
        return selector;
      } catch {
        // Try next selector
      }
    }
    this.log.warn({ selectors: JOB_CARD_SELECTORS }, 'All job card selectors timed out');
    return null;
  }

  /**
   * Find job cards using the first matching selector.
   */
  private async findJobCards(page: Page): Promise<Locator[]> {
    for (const selector of JOB_CARD_SELECTORS) {
      const cards = await page.locator(selector).all();
      if (cards.length > 0) return cards;
    }
    return [];
  }

  /**
   * Find the URL of the next page of results.
   * Returns null if no next page exists.
   */
  private async findNextPageUrl(page: Page, currentUrl: string): Promise<string | null> {
    try {
      // Indeed uses a "Next" link with aria-label="Next Page" or similar
      const nextLink = page.locator('a[aria-label*="Next"], a[aria-label*="next"]').first();
      const exists = await nextLink.count();
      if (exists === 0) return null;

      const href = await nextLink.getAttribute('href');
      if (!href) return null;

      // Make relative URLs absolute
      return new URL(href, currentUrl).href;
    } catch {
      return null;
    }
  }

  // ── Job extraction ──────────────────────────────────────────────────

  /**
   * Extract job data from a card.
   * Uses DOM extraction for deterministic fields (title, company, location).
   * LLM only for requirements, salary, remote type.
   */
  private async extractJobFromCard(card: Locator, page: Page): Promise<ScrapedJob | null> {
    try {
      // Safer click — scroll into view first
      await card.scrollIntoViewIfNeeded();
      await card.click({ timeout: CLICK_TIMEOUT_MS });

      // Wait for detail panel OR navigation to job page
      const jobDetail = page.locator('[data-testid="job-detail"], .jobsearch-ViewJobPane-content').first();
      try {
        await jobDetail.waitFor({ state: 'visible', timeout: DETAIL_TIMEOUT_MS });
      } catch {
        // Detail panel didn't appear - may have navigated to standalone job page
        // Continue and try to extract from current page
      }

      // ── Deterministic DOM extraction (title, company, location) ──
      const title = await this.extractText(page, [
        '[data-testid="job-title"]',
        '.jobsearch-JobInfoHeader-title',
        'h1',
      ]);
      const company = await this.extractText(page, [
        '[data-testid="company-name"]',
        '.companyName',
        '[data-company-name]',
      ]);
      const companyUrl = await this.extractCompanyUrl(page);
      const location = await this.extractText(page, [
        '[data-testid="company-location"]',
        '.companyLocation',
        '[data-geo-location]',
      ]);
      const jobUrl = page.url();

      if (!title) return null;

      // ── LLM enrichment (requirements, salary, remote type) ──
      // Target only the job description container to avoid sidebar noise
      const descriptionText = await this.extractJobDescriptionText(page);
      const extracted = await this.extractWithLLM(descriptionText);

      // Stable ID via SHA-256 hash using unit separator (non-printable)
      const externalId = `indeed-${crypto.createHash('sha256').update(`${title}${HASH_DELIMITER}${company}${HASH_DELIMITER}${location}`).digest('hex').slice(0, 16)}`;

      // Extract apply link — may redirect to external ATS
      const applicationUrl = await this.extractApplyUrl(page);

      return {
        external_id: externalId,
        title,
        company: company || 'Unknown',
        location: location || 'Unknown',
        remote_type: extracted.remote_type || this.inferRemoteType(location),
        salary_min: extracted.salary_min || 0,
        salary_max: extracted.salary_max || 0,
        salary_currency: extracted.salary_currency || 'USD',
        description: extracted.description || descriptionText,
        requirements: extracted.requirements || '',
        url: jobUrl,
        application_url: applicationUrl || jobUrl,
        company_url: companyUrl,
        source: 'indeed',
      };
    } catch (err) {
      this.log.error({ err }, 'Job extraction failed');
      return null;
    }
  }

  /**
   * Extract text from the job description container specifically.
   * Falls back to full detail pane if description container not found.
   */
  private async extractJobDescriptionText(page: Page): Promise<string> {
    for (const selector of JOB_DESCRIPTION_SELECTORS) {
      try {
        const text = await page.locator(selector).first().innerText({ timeout: 1000 });
        if (text.trim()) return text;
      } catch {
        // Try next selector
      }
    }

    // Fallback: try the detail panel
    try {
      return await page.locator('[data-testid="job-detail"], .jobsearch-ViewJobPane-content').first().innerText();
    } catch {
      return '';
    }
  }

  /**
   * Extract the apply URL from the job detail page.
   * Indeed apply buttons may link to external ATS (Greenhouse, Lever, etc.)
   * or open an Indeed-hosted form. Returns the href of the apply link.
   */
  private async extractApplyUrl(page: Page): Promise<string> {
    const applySelectors = [
      'a[id="apply-button-link"]',
      'a.apply-button',
      'button[id="apply-button"] a',
      'a:has-text("Apply now")',
      'a:has-text("Apply Now")',
      'a:has-text("Apply on company site")',
    ];

    for (const selector of applySelectors) {
      try {
        const link = page.locator(selector).first();
        // Check if element exists first
        const count = await link.count();
        if (count === 0) continue;

        const href = await link.getAttribute('href');
        if (href && (href.startsWith('http') || href.startsWith('//'))) {
          return href.startsWith('//') ? `https:${href}` : href;
        }
      } catch {
        // Try next selector
      }
    }

    return '';
  }

  /**
   * Extract the company URL from the job detail page.
   * Indeed job cards often have a company link.
   */
  private async extractCompanyUrl(page: Page): Promise<string> {
    try {
      const link = page.locator('[data-testid="company-name"] a, .companyName a').first();
      const count = await link.count();
      if (count === 0) return '';

      const href = await link.getAttribute('href');
      if (href && (href.startsWith('http') || href.startsWith('//'))) {
        return href.startsWith('//') ? `https:${href}` : href;
      }
    } catch {
      // Ignore
    }
    return '';
  }

  /**
   * Extract text from the first matching element.
   * Uses an array of selectors (more robust than comma-separated string).
   */
  private async extractText(page: Page, selectors: readonly string[]): Promise<string> {
    for (const sel of selectors) {
      try {
        const text = await page.locator(sel).first().innerText({ timeout: 1000 });
        if (text.trim()) return text.trim();
      } catch {
        // Try next selector
      }
    }
    return '';
  }

  /**
   * Infer remote type from location string.
   */
  private inferRemoteType(locationName: string): 'remote' | 'hybrid' | 'onsite' | 'unknown' {
    const lower = locationName.toLowerCase();
    if (lower.includes('remote')) return 'remote';
    if (lower.includes('hybrid')) return 'hybrid';
    if (lower === 'unknown' || lower === '') return 'unknown';
    return 'onsite';
  }
}