import { BrowserContext, Page } from 'playwright';
import { writeFile, mkdir } from 'node:fs/promises';
import { join } from 'node:path';
import { detectFormFields } from './detector.js';
import { mapFieldsToCandidateData } from './fields.js';
import { fillForm, FillResult, submitForm } from './submitter.js';
import { getBrowser } from '../utils/browser.js';
import { stealthConfig } from '../utils/stealth.js';
import { logger } from '../utils/logger.js';

const log = logger.child({ component: 'FormFiller' });

/**
 * Options for the form filler orchestrator.
 */
export interface FormFillOptions {
  /** Target URL to fill (must be http/https). */
  url: string;
  /** Candidate profile data (name, email, experience, etc.). */
  candidateData: Record<string, unknown>;
  /** Optional file path to a resume PDF for file upload fields. */
  resumePath?: string;
  /** Optional file path to a cover letter PDF for file upload fields. */
  coverLetterPath?: string;
  /** Optional file path to a portfolio file for file upload fields. */
  portfolioPath?: string;
  /** Optional CSS selector to wait for before detecting fields. */
  waitForSelector?: string;
  /** Navigation timeout in ms. @default 60000 */
  navigationTimeoutMs?: number;
  /** Selector wait timeout in ms. @default 10000 */
  selectorTimeoutMs?: number;
  /** Optional AbortSignal for cancellation. */
  signal?: AbortSignal;
  /** Whether LLM mapping was used (for monitoring). */
  usedLLM?: boolean;
}

// ── SSRF protection ────────────────────────────────────────────────

/** Blocked URL schemes. */
const BLOCKED_SCHEMES = ['file:', 'data:', 'javascript:'];

/** Exact hostnames that are always blocked (after normalization). */
const BLOCKED_EXACT_HOSTS = new Set([
  'localhost', '127.0.0.1', '0.0.0.0', '::1',
  'metadata.google.internal', 'instance-data',
  'metadata.azure.com', '169.254.169.254',
  'metadata.digitalocean.com',
]);

/** Private IPv4 ranges (10/8, 172.16/12, 192.168/16). */
const PRIVATE_IP_REGEX = /^(10\.|172\.(1[6-9]|2\d|3[01])\.|192\.168\.)/;

/** Private IPv6 ranges (ULA fc00::/7, link-local fe80::/10). */
const PRIVATE_IPV6_REGEX = /^(fc|fd|fe8)/i;

/** Detect encoded IP forms like decimal (2130706433 = 127.0.0.1) or hex. */
const ENCODED_IP_REGEX = /^(0x[0-9a-f]+|\d{8,12})$/;

function isBlockedHost(hostname: string): boolean {
  // Strip brackets, extract IPv4-mapped IPv6 (e.g. ::ffff:192.168.1.1)
  let h = hostname.toLowerCase().replace(/^\[|\]$/g, '');
  const ipv4InV6 = h.match(/(?:ffff:)(\d+\.\d+\.\d+\.\d+)$/);
  if (ipv4InV6) h = ipv4InV6[1];

  if (BLOCKED_EXACT_HOSTS.has(h)) return true;
  if (h.startsWith('169.254.')) return true;
  if (PRIVATE_IP_REGEX.test(h)) return true;
  if (PRIVATE_IPV6_REGEX.test(h)) return true;
  if (ENCODED_IP_REGEX.test(h)) return true;
  if (/^0\d{2,}\./.test(h)) return true; // octal IPv4 (e.g. 0177.0.0.1 = 127.0.0.1)
  if (h.endsWith('.localhost')) return true;
  return false;
}

/**
 * Validate URL against SSRF protections.
 * Throws if URL is blocked.
 */
function validateUrl(url: string): void {
  const parsed = new URL(url);
  if (BLOCKED_SCHEMES.includes(parsed.protocol) || isBlockedHost(parsed.hostname)) {
    throw new Error(`Blocked navigation to restricted URL: ${parsed.origin}`);
  }
}

// ── Orchestrator ───────────────────────────────────────────────────

/**
 * Orchestrates form detection, field mapping, filling, and submission
 * for a job application page using Playwright.
 *
 * Reuses the shared Chromium browser singleton from `utils/browser.ts`.
 * Each call creates a fresh BrowserContext (isolation) and closes it
 * on completion.
 *
 * @param options - URL, candidate data, and optional selectors.
 * @returns FillResult indicating success and any field-level errors.
 */
export async function fillApplicationForm(options: FormFillOptions): Promise<FillResult> {
  const started = Date.now();

  // ── Input validation ────────────────────────────────────────────
  if (Object.keys(options.candidateData).length === 0) {
    throw new Error('candidateData cannot be empty');
  }

  // ── SSRF protection ─────────────────────────────────────────────
  validateUrl(options.url);

  // ── Cancellation check ──────────────────────────────────────────
  options.signal?.throwIfAborted();

  log.info({ url: options.url }, 'Starting form fill');

  const browser = await getBrowser();
  let context: BrowserContext | undefined;
  let page: Page | undefined;
  let screenshotOnFailure = true;

  try {
    context = await browser.newContext(stealthConfig);
    page = await context.newPage();

    // Navigate — use domcontentloaded (faster, no WebSocket/SSE hangs)
    await page.goto(options.url, {
      waitUntil: 'domcontentloaded',
      timeout: options.navigationTimeoutMs ?? 60000,
    });

    // Check for cancellation after navigation
    options.signal?.throwIfAborted();

    // Wait for form if specified — log warning on timeout
    if (options.waitForSelector) {
      try {
        await page.waitForSelector(options.waitForSelector, {
          timeout: options.selectorTimeoutMs ?? 10000,
        });
      } catch {
        log.warn(
          { selector: options.waitForSelector },
          'waitForSelector timed out',
        );
      }
    }

    // Check for cancellation before field detection
    options.signal?.throwIfAborted();

    // Detect form fields
    const fields = await detectFormFields(page);
    if (fields.length === 0) {
      // Capture screenshot before throwing
      try {
        const ss = await page.screenshot({ fullPage: true, type: 'png' });
        const ssPath = await saveScreenshot(ss, 'no-fields');
        log.error({ url: options.url, screenshotPath: ssPath }, 'No form fields detected');
      } catch {
        log.error({ url: options.url }, 'No form fields detected (screenshot failed)');
      }
      throw new Error('No form fields detected on page');
    }
    log.info({ fieldCount: fields.length }, 'Detected form fields');

    // Check for cancellation before LLM mapping
    options.signal?.throwIfAborted();

    // Map fields to candidate data using LLM
    const mappings = await mapFieldsToCandidateData(fields, options.candidateData);
    log.info({ mappingCount: mappings.length }, 'Mapped fields to candidate data');

    // Fill the form with smart file upload
    const result = await fillForm(page, fields, mappings, {
      resumePath: options.resumePath,
      coverLetterPath: options.coverLetterPath,
      portfolioPath: options.portfolioPath,
    });

    // Submit the form
    if (result.success) {
      const urlBeforeSubmit = page.url();
      const submitted = await submitForm(page);
      if (!submitted) {
        result.errors.push('Could not find submit button');
        result.success = false;
        // Capture screenshot for debugging
        try {
          const ss = await page.screenshot({ fullPage: true, type: 'png' });
          const ssPath = await saveScreenshot(ss, 'submit-not-found');
          log.warn({ url: page.url(), screenshotPath: ssPath }, 'Submit button not found');
        } catch { /* best effort */ }
      } else {
        // Post-submit verification: check if URL changed
        try {
          await page.waitForURL(url => url.toString() !== urlBeforeSubmit, { timeout: 5000 });
          log.info({ url: page.url() }, 'Page navigated after submit');
        } catch {
          // URL didn't change — might still be fine (AJAX submit, same-page validation)
          log.warn({ url: page.url() }, 'Page URL did not change after submit');
        }
      }
    }

    screenshotOnFailure = false; // Success path — no need for failure screenshot
    return result;
  } catch (error) {
    // Capture screenshot on failure for debugging
    if (screenshotOnFailure && page) {
      try {
        const ss = await page.screenshot({ fullPage: true, type: 'png' });
        const ssPath = await saveScreenshot(ss, 'form-fill-failed');
        log.error(
          { url: options.url, durationMs: Date.now() - started, screenshotPath: ssPath },
          'Form fill failed',
        );
      } catch {
        log.error(
          { url: options.url, durationMs: Date.now() - started },
          'Form fill failed (screenshot failed)',
        );
      }
    }
    throw error;
  } finally {
    // Explicitly close page first for clean teardown
    await page?.close().catch(() => {});
    await context?.close().catch(() => {});
    log.info(
      { url: options.url, durationMs: Date.now() - started, success: !screenshotOnFailure },
      'Form fill completed',
    );
  }
}

/**
 * Save a screenshot to the configured storage directory.
 * Returns the file path.
 */
async function saveScreenshot(buffer: Buffer, prefix: string): Promise<string> {
  const storageDir = process.env.STORAGE_DIR ?? 'storage';
  const screenshotsDir = join(storageDir, 'screenshots');
  await mkdir(screenshotsDir, { recursive: true });
  const filename = `${prefix}-${Date.now()}.png`;
  const filepath = join(screenshotsDir, filename);
  await writeFile(filepath, buffer);
  return filepath;
}

// Re-export for external use
export { FillResult, submitForm } from './submitter.js';