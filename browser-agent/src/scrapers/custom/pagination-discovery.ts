import { Page } from 'playwright';
import { discoverJobLinks } from './link-discovery.js';
import { logger } from '../../utils/logger.js';

const log = logger.child({ component: 'PaginationDiscovery' });

const MAX_PAGINATION_PAGES = 10;

// Patterns passed as strings to page.evaluate (RegExp not serializable)
const PAGINATION_TEXT_PATTERN_SOURCE = 'next|›|»|→|>\\s*\\d+';
const PAGINATION_HREF_PATTERN_SOURCE = '[?&]page=\\d+|[?&]p=\\d+|/page/\\d+|[?&]offset=\\d+';

/**
 * Discover pagination links (next page, numbered pages).
 * @param pageUrl Current page URL (used as base for relative link resolution)
 */
async function discoverPaginationLinks(page: Page, pageUrl: string): Promise<string[]> {
  const links = await page.evaluate(
    ({ textPatternSource, hrefPatternSource }: { textPatternSource: string; hrefPatternSource: string }): string[] => {
      const results: string[] = [];
      const textPattern = new RegExp(textPatternSource, 'i');
      const hrefPattern = new RegExp(hrefPatternSource, 'i');
      const anchors = Array.from(document.querySelectorAll<HTMLAnchorElement>('a[href]'));

      for (const a of anchors) {
        const href = a.href;
        const text = (a.textContent ?? '').trim();

        // Match by link text (Next, ›, », →)
        if (textPattern.test(text)) {
          results.push(href);
          continue;
        }

        // Match by href pattern (?page=2, /page/2, ?offset=20)
        if (hrefPattern.test(href)) {
          results.push(href);
        }
      }

      return results;
    },
    { textPatternSource: PAGINATION_TEXT_PATTERN_SOURCE, hrefPatternSource: PAGINATION_HREF_PATTERN_SOURCE }
  );

  // Make absolute and deduplicate using pageUrl as base
  const seen = new Set<string>();
  const absolute: string[] = [];

  for (const link of links) {
    try {
      const url = new URL(link, pageUrl).href;
      if (!seen.has(url) && url !== pageUrl) {
        seen.add(url);
        absolute.push(url);
      }
    } catch (err) {
      log.warn({ href: link, pageUrl, err }, 'Failed to normalize pagination URL');
    }
  }

  return absolute;
}

/**
 * Discover job links across multiple pages by following pagination.
 * Returns deduplicated list of job page URLs.
 */
export async function discoverWithPagination(
  page: Page,
  baseUrl: string,
  maxJobLinks: number = 50
): Promise<string[]> {
  const seen = new Set<string>();
  const pagesToVisit: string[] = [baseUrl];
  const pendingPages = new Set<string>([baseUrl]);
  const visitedPages = new Set<string>();

  while (pagesToVisit.length > 0 && visitedPages.size < MAX_PAGINATION_PAGES) {
    const pageUrl = pagesToVisit.shift()!;
    pendingPages.delete(pageUrl);
    if (visitedPages.has(pageUrl)) continue;
    visitedPages.add(pageUrl);

    try {
      if (pageUrl !== baseUrl) {
        await page.goto(pageUrl, {
          waitUntil: 'domcontentloaded',
          timeout: 30_000,
        });
        // Skip autoScroll on pagination pages — they're listing pages
        // without lazy-loaded content; only baseUrl needs scrolling
      }

      // Score and collect job links from this page using pageUrl as base
      const scored = await discoverJobLinks(page, pageUrl, maxJobLinks);
      for (const s of scored) {
        if (!seen.has(s)) {
          seen.add(s);
        }
      }

      // Discover pagination links from current page
      const nextPages = await discoverPaginationLinks(page, pageUrl);
      for (const np of nextPages) {
        if (!visitedPages.has(np) && !pendingPages.has(np)) {
          pagesToVisit.push(np);
          pendingPages.add(np);
        }
      }
    } catch (err) {
      log.warn({ err, pageUrl }, 'Failed to visit pagination page');
    }
  }

  log.info({ pagesVisited: visitedPages.size, linksFound: seen.size }, 'Pagination discovery complete');
  return Array.from(seen);
}