import { BaseScraper } from '../base.js';
import { logger } from '../../utils/logger.js';

const log = logger.child({ component: 'RedirectResolver' });

/**
 * Resolve a URL through any redirects.
 * Uses fetch() first (lightweight, follows HTTP redirects),
 * falls back to browser navigation for JS-based redirects.
 */
export async function resolveRedirect(url: string, scraper: BaseScraper): Promise<string> {
  // Fast path: HTTP redirect following via fetch
  try {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 10_000);

    const response = await fetch(url, {
      method: 'HEAD',
      redirect: 'follow',
      signal: controller.signal,
    });
    clearTimeout(timer);

    // response.url gives the final URL after redirects
    if (response.url && response.url !== 'about:blank') {
      log.debug({ original: url, resolved: response.url }, 'Resolved redirect via fetch');
      return response.url;
    }
  } catch {
    // fetch failed (CORS, network error) — fall through to browser
  }

  // Slow path: browser navigation for JS-based redirects
  const page = await scraper.createPage();
  try {
    const response = await page.goto(url, {
      waitUntil: 'domcontentloaded',
      timeout: 10_000,
    });

    if (response?.url() && response.url() !== 'about:blank') {
      log.debug({ original: url, resolved: response.url() }, 'Resolved redirect via browser (response.url)');
      return response.url();
    }

    const finalUrl = page.url();
    log.debug({ original: url, resolved: finalUrl }, 'Resolved redirect via browser (page.url)');
    return finalUrl;
  } catch {
    log.warn({ original: url }, 'Redirect resolution failed, returning original');
    return url;
  } finally {
    await page.close().catch(() => {});
  }
}