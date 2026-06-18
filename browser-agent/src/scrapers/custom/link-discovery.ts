import { Page } from 'playwright';
import { logger } from '../../utils/logger.js';

const log = logger.child({ component: 'LinkDiscovery' });

/**
 * Keywords in link text that strongly indicate a job posting.
 * Scored additively — higher = more likely a job link.
 */
export const JOB_TEXT_KEYWORDS = [
  'engineer', 'developer', 'manager', 'designer', 'analyst',
  'scientist', 'architect', 'lead', 'senior', 'junior',
  'staff', 'principal', 'director', 'head', 'vp',
  'specialist', 'coordinator', 'consultant', 'intern',
  'associate', 'partner', 'officer', 'administrator',
];

/** Minimum text length to consider a link as having meaningful text. */
const MIN_JOB_TITLE_LENGTH = 10;

/** Text length indicating a good job title. */
const GOOD_JOB_TITLE_LENGTH = 20;

/** Maximum contribution from text keyword matches to prevent domination. */
const MAX_TEXT_SCORE = 6;

/** URL patterns with their associated scores for job link detection. */
const JOB_URL_PATTERNS: Array<{ pattern: RegExp; score: number }> = [
  { pattern: /\/job[s]?\//i, score: 3 },
  { pattern: /\/career[s]?\//i, score: 3 },
  { pattern: /\/position[s]?\//i, score: 3 },
  { pattern: /\/opening[s]?\//i, score: 3 },
  { pattern: /\/vacancy/i, score: 3 },
  { pattern: /\/posting[s]?\//i, score: 3 },
  { pattern: /\/role[s]?\//i, score: 3 },
  { pattern: /\/opportunity/i, score: 3 },
  { pattern: /\/apply/i, score: 2 },
  { pattern: /\/detail[s]?\//i, score: 2 },
  // Numeric ID in URL path (common ATS pattern: /jobs/123456)
  { pattern: /\/jobs?\/\d{3,}(?:\/|$)/i, score: 2 },
  { pattern: /\/careers?\/\d{3,}(?:\/|$)/i, score: 2 },
];

/** Type for the page.evaluate callback arguments. */
interface ScoreJobLinksArgs {
  textKeywords: readonly string[];
  selector: string;
}

/** Type for the page.evaluate return value. */
interface ScoredLink {
  href: string;
  score: number;
}

/**
 * Score all links on the page by href pattern + link text.
 * Returns scored links sorted by score descending.
 */
export async function scoreJobLinks(
  page: Page,
  baseUrl: string,
  maxLinks: number = 50
): Promise<Array<{ href: string; score: number }>> {
  try {
    const scored = await page.evaluate(
      ({ textKeywords, selector }: ScoreJobLinksArgs): ScoredLink[] => {
        const links = Array.from(document.querySelectorAll<HTMLAnchorElement>(selector));
        const scored: ScoredLink[] = [];

        for (const a of links) {
          const href = a.href;
          const text = (a.textContent ?? '').trim().toLowerCase();
          let score = 0;

          // URL pattern scoring
          for (const { pattern, score: patternScore } of JOB_URL_PATTERNS) {
            if (pattern.test(href)) score += patternScore;
          }

          // Text content scoring (capped to prevent domination)
          let textMatches = 0;
          for (const kw of textKeywords) {
            if (text.includes(kw)) textMatches++;
          }
          score += Math.min(textMatches * 2, MAX_TEXT_SCORE);

          // Length heuristic: very short text = likely icon/link, not job title
          if (text.length > MIN_JOB_TITLE_LENGTH) score += 1;
          if (text.length > GOOD_JOB_TITLE_LENGTH) score += 1;

          // Skip navigation/utility links
          if (/^(#|javascript:|mailto:|tel:)/i.test(href)) continue;
          if (score > 0) {
            scored.push({ href, score });
          }
        }

        return scored;
      },
      { textKeywords: JOB_TEXT_KEYWORDS, selector: 'a[href]' }
    );

    log.debug({ baseUrl, linksFound: scored.length, maxLinks }, 'Job link scoring complete');

    // Sort by score descending
    return scored
      .sort((a, b) => b.score - a.score)
      .slice(0, maxLinks);
  } catch (err) {
    log.error({ baseUrl, err }, 'Failed to score job links');
    return [];
  }
}

/**
 * Discover job-related links from the listing page.
 * Makes relative URLs absolute and deduplicates.
 */
export async function discoverJobLinks(
  page: Page,
  baseUrl: string,
  maxLinks: number = 50
): Promise<string[]> {
  const scored = await scoreJobLinks(page, baseUrl, maxLinks);

  const seen = new Set<string>();
  const absolute: string[] = [];

  for (const { href } of scored) {
    try {
      const url = new URL(href, baseUrl).href;
      if (!seen.has(url)) {
        seen.add(url);
        absolute.push(url);
      }
    } catch (err) {
      log.warn({ href, baseUrl, err }, 'Failed to normalize URL');
    }
  }

  log.debug({ baseUrl, discoveredLinks: absolute.length }, 'Job link discovery complete');
  return absolute;
}