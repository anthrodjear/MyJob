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
  // Check if existing browser is still connected
  if (sharedBrowser?.isConnected()) return sharedBrowser;

  // If browser was disconnected, reset and create new one
  if (sharedBrowser) {
    sharedBrowser = null;
  }

  if (browserPromise) return browserPromise;

  const headless = process.env.BROWSER_HEADLESS !== 'false'; // Default true, allow override
  const noSandbox = process.env.BROWSER_NO_SANDBOX === 'true'; // Opt-in for Docker

  const launchArgs = [
    '--disable-blink-features=AutomationControlled',
    '--disable-dev-shm-usage',
    '--disable-gpu',
    '--disable-extensions',
    '--disable-background-networking',
    '--disable-background-timer-throttling',
    '--disable-renderer-backgrounding',
    '--no-first-run',
    '--no-default-browser-check',
  ];

  if (noSandbox) {
    launchArgs.push('--no-sandbox');
  }

  browserPromise = chromium.launch({
    headless,
    args: launchArgs,
  }).then(browser => {
    sharedBrowser = browser;
    browserPromise = null;
    logger.info({ pid: process.pid, headless, noSandbox }, 'Chromium browser launched');
    return browser;
  }).catch(err => {
    browserPromise = null;
    logger.error({ err }, 'Failed to launch Chromium browser');
    throw err;
  });

  return browserPromise;
}

/**
 * Close the shared browser instance with a timeout.
 * Call on process shutdown.
 */
export async function closeBrowser(): Promise<void> {
  if (sharedBrowser) {
    const browser = sharedBrowser;
    sharedBrowser = null;

    try {
      await Promise.race([
        browser.close(),
        new Promise<void>((_, reject) => setTimeout(() => reject(new Error('Browser close timeout')), 10_000)),
      ]);
      logger.info({ pid: process.pid }, 'Chromium browser closed');
    } catch (err) {
      logger.error({ err }, 'Error closing Chromium browser');
    }
  }
}

/**
 * Reset browser state for testing.
 * Only use in test environments.
 */
export function resetBrowserForTesting(): void {
  sharedBrowser = null;
  browserPromise = null;
}