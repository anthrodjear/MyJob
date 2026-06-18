import { Page } from 'playwright';
import { logger } from '../../utils/logger.js';

const log = logger.child({ component: 'ContentExtractor' });

/** Selectors for elements to remove during content extraction. */
export const CONTENT_REMOVE_SELECTORS = [
  'script', 'style', 'noscript', 'header', 'footer', 'nav', 'aside',
] as const;

/** Maximum number of links to include in extracted content. */
const MAX_LINKS = 500;

/** Scroll step in pixels. */
const SCROLL_STEP_PX = 300;

/** Scroll interval in milliseconds. */
const SCROLL_INTERVAL_MS = 150;

/** Maximum total scroll distance in pixels. */
const MAX_SCROLL_PX = 15_000;

/**
 * Extract cleaned page content via browser-side JavaScript.
 * Removes noise elements (scripts, styles, nav, footer, etc.)
 * and returns inner text + link inventory.
 *
 * @param page - Playwright Page instance
 * @returns Formatted string with TITLE, CONTENT, and LINKS sections
 */
export async function extractPageContent(page: Page): Promise<string> {
  try {
    return await page.evaluate(({ removeSelectors, maxLinks }) => {
      // Remove noise elements
      for (const sel of removeSelectors) {
        document.querySelectorAll(sel).forEach(el => el.remove());
      }

      // Get visible text (innerText respects CSS visibility)
      const body = document.body;
      if (!body) return '';

      const text = body.innerText || '';

      // Collect link inventory (text -> href)
      const links = Array.from(document.querySelectorAll<HTMLAnchorElement>('a[href]'))
        .slice(0, maxLinks)
        .map(a => {
          const txt = a.textContent?.trim() ?? '';
          return txt ? `${txt} -> ${a.href}` : a.href;
        })
        .filter(Boolean)
        .join('\n');

      return `
TITLE: ${document.title}

CONTENT:
${text}

LINKS:
${links}
      `.trim();
    }, { removeSelectors: CONTENT_REMOVE_SELECTORS, maxLinks: MAX_LINKS });
  } catch (err) {
    log.error({ err, url: page.url() }, 'Failed to extract page content');
    throw err;
  }
}

/**
 * Auto-scroll the page to trigger lazy-loaded content and infinite scroll.
 * Scrolls in steps, stops after max distance or page end.
 * Waits for network idle after scrolling to allow lazy content to load.
 *
 * @param page - Playwright Page instance
 */
export async function autoScroll(page: Page): Promise<void> {
  try {
    await page.evaluate(async ({ step, interval, maxScroll }) => {
      if (!document.body || document.body.scrollHeight === 0) return;

      await new Promise<void>(resolve => {
        let total = 0;

        const timer = setInterval(() => {
          window.scrollBy(0, step);
          total += step;

          if (total >= document.body.scrollHeight || total >= maxScroll) {
            clearInterval(timer);
            resolve();
          }
        }, interval);
      });
    }, { step: SCROLL_STEP_PX, interval: SCROLL_INTERVAL_MS, maxScroll: MAX_SCROLL_PX });

    // Wait for any lazy-loaded content to appear after scrolling
    await page.waitForLoadState('networkidle', { timeout: 10_000 }).catch(() => {
      // Ignore timeout - page may not have network activity
    });
  } catch (err) {
    log.warn({ err, url: page.url() }, 'Auto-scroll failed');
    // Don't throw - scroll failure shouldn't block extraction
  }
}