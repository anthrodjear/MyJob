import { chromium, Browser } from 'playwright';
import { logger } from './logger.js';

/**
 * Shared Chromium browser singleton.
 *
 * All modules (scrapers, form filler) should use `getBrowser()` instead of
 * launching their own instances. This avoids spawning multiple Chromium
 * processes and centralizes lifecycle management.
 *
 * Call `closeBrowser()` on process shutdown.
 */

let sharedBrowser: Browser | null = null;
let browserPromise: Promise<Browser> | null = null;

/**
 * Get or launch the shared Chromium browser instance.
 * Race-condition safe — concurrent callers get the same promise.
 */
export async function getBrowser(): Promise<Browser> {
  if (sharedBrowser?.isConnected()) return sharedBrowser;
  if (browserPromise) return browserPromise;

  browserPromise = chromium.launch({
    headless: true,
    args: ['--disable-blink-features=AutomationControlled', '--disable-dev-shm-usage'],
  }).then(browser => {
    sharedBrowser = browser;
    browserPromise = null;
    logger.info({}, 'Chromium browser launched');
    return browser;
  }).catch(err => {
    browserPromise = null;
    throw err;
  });

  return browserPromise;
}

/**
 * Close the shared browser instance with a 5s timeout.
 * Call on process shutdown.
 */
export async function closeBrowser(): Promise<void> {
  if (sharedBrowser) {
    const browser = sharedBrowser;
    sharedBrowser = null;
    await Promise.race([
      browser.close(),
      new Promise<void>(r => setTimeout(r, 5000)),
    ]).catch(() => {});
    logger.info({}, 'Chromium browser closed');
  }
}
