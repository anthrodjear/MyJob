import { Page } from 'playwright';
import { ScrapedJob } from '../base.js';
import { scrapeGreenhouse } from '../greenhouse.js';
import { scrapeLever } from '../lever.js';
import { logger } from '../../utils/logger.js';
import { detectSourceFromUrl } from './helpers.js';

const log = logger.child({ component: 'ATSDetector' });

/**
 * Known ATS scrapers that we can delegate to.
 * Each entry maps detection patterns to the scraper function.
 *
 * Adding a new ATS scraper: add an entry here + update `sourceLabel` in helpers.ts
 * for `detectSourceFromUrl` to return the correct identifier.
 */
export interface ATSScraper {
  name: string;
  scraper: (url: string) => Promise<ScrapedJob[]>;
  patterns: RegExp[];
}

const ATS_SCRAPERS: ATSScraper[] = [
  {
    name: 'greenhouse',
    scraper: scrapeGreenhouse,
    patterns: [/greenhouse\.io/i],
  },
  {
    name: 'lever',
    scraper: scrapeLever,
    patterns: [/lever\.co/i],
  },
];

/**
 * Detect a known ATS from URL or page HTML.
 * Returns the dedicated scraper if supported, null otherwise.
 *
 * Detection strategy:
 *   1. Check URL patterns (cheapest)
 *   2. Check page HTML <head> only (avoid false positives from job descriptions)
 *      e.g. a Greenhouse page mentioning Lever in a job description would otherwise
 *      match Lever patterns and route incorrectly.
 */
export async function detectATS(page: Page, baseUrl: string): Promise<ATSScraper | null> {
  // Step 1: Check URL patterns first (cheapest, most reliable)
  for (const ats of ATS_SCRAPERS) {
    for (const pattern of ats.patterns) {
      if (pattern.test(baseUrl)) {
        log.info({ baseUrl, ats: ats.name, source: 'url' }, 'Detected known ATS');
        return ats;
      }
    }
  }

  // Step 2: Check page HTML <head> only (avoid job description false positives)
  try {
    // Use timeout to prevent hangs on slow/unresponsive pages
    const html = await Promise.race([
      page.content(),
      new Promise<never>((_, reject) =>
        setTimeout(() => reject(new Error('page.content() timed out after 5s')), 5_000)
      ),
    ]);

    // Extract only the <head> section to avoid job descriptions and sidebar links
    const headMatch = html.match(/<head[^>]*>([\s\S]*?)<\/head>/i);
    const headHtml = headMatch ? headMatch[1] : html;

    for (const ats of ATS_SCRAPERS) {
      for (const pattern of ats.patterns) {
        if (pattern.test(headHtml)) {
          log.info({ baseUrl, ats: ats.name, source: 'html_head' }, 'Detected known ATS');
          return ats;
        }
      }
    }
  } catch (err) {
    log.debug({ err, baseUrl }, 'Failed to fetch page HTML for ATS detection');
    // Continue with null — caller will use generic scraping
  }

  return null;
}

// Re-export detectSourceFromUrl from helpers.ts to maintain single source of truth
export { detectSourceFromUrl };