import { BrowserContext, Page } from 'playwright';
import { getOllamaClient, JobExtractionResult } from '../llm/ollama.js';
import { getBrowser } from '../utils/browser.js';
import { stealthConfig } from '../utils/stealth.js';
import { logger } from '../utils/logger.js';

// Type declaration for browser globals in Playwright initScript context
declare const _navigator: Navigator;

/**
 * Normalized representation of a scraped job posting.
 */
export interface ScrapedJob {
  external_id: string;
  title: string;
  company: string;
  location: string;
  remote_type: string;
  salary_min: number;
  salary_max: number;
  salary_currency: string;
  description: string;
  requirements: string;
  /** Source listing URL (provenance, dedup). */
  url: string;
  /** Actual application URL — where the user applies. Resolved from apply buttons/redirects. */
  application_url: string;
  company_url: string;
  /** Scraper source identifier: 'greenhouse', 'lever', 'remoteok', 'indeed', 'custom'. */
  source: string;
}

// ── Stealth scripts ────────────────────────────────────────────────

const STEALTH_SCRIPTS = `
  Object.defineProperty(navigator, 'webdriver', { get: () => false });
  Object.defineProperty(navigator.__proto__, 'webdriver', { get: () => false });

  // Permissions API
  const originalQuery = window.navigator.permissions.query;
  window.navigator.permissions.query = (parameters) => (
    parameters.name === 'notifications' ?
      Promise.resolve({ state: Notification.permission }) :
      originalQuery(parameters)
  );

  // Realistic plugins (no hardcoded extension IDs)
  Object.defineProperty(navigator, 'plugins', {
    get: () => {
      const arr = [
        { name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer' },
        { name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai' },
        { name: 'Native Client', filename: 'internal-nacl-plugin' },
      ];
      Object.setPrototypeOf(arr, PluginArray.prototype);
      return arr;
    },
  });

  Object.defineProperty(navigator, 'languages', { get: () => ['en-US', 'en'] });

  // Hardware concurrency & device memory
  Object.defineProperty(navigator, 'hardwareConcurrency', { get: () => 8 });
  Object.defineProperty(navigator, 'deviceMemory', { get: () => 8 });

  // Connection API
  Object.defineProperty(navigator, 'connection', {
    get: () => ({
      effectiveType: '4g',
      rtt: 50,
      downlink: 10,
      saveData: false,
    }),
  });

  // Screen properties
  Object.defineProperty(screen, 'colorDepth', { get: () => 24 });
  Object.defineProperty(screen, 'pixelDepth', { get: () => 24 });

  if (!window.chrome) { window.chrome = { runtime: {} }; }

  // WebGL vendor/renderer spoofing
  const getParameter = WebGLRenderingContext.prototype.getParameter;
  WebGLRenderingContext.prototype.getParameter = function(parameter) {
    if (parameter === 37445) return 'Intel Inc.'; // UNMASKED_VENDOR_WEBGL
    if (parameter === 37446) return 'Intel Iris OpenGL Engine'; // UNMASKED_RENDERER_WEBGL
    return getParameter.call(this, parameter);
  };
`;

// ── Base class ─────────────────────────────────────────────────────

/**
 * Base class for all job site scrapers.
 *
 * Provides Playwright browser lifecycle management (shared singleton from
 * `utils/browser.ts`), stealth configuration, LLM-based extraction, and
 * retry logic. Subclasses implement {@link scrape}.
 *
 * @example
 *   class MyScraper extends BaseScraper {
 *     async scrape(baseUrl: string) { ... }
 *   }
 */
export abstract class BaseScraper {
  protected ollama = getOllamaClient();

  /** Logger instance — injected via constructor for testability. */
  protected readonly log = logger.child({ component: 'BaseScraper' });

  /** Tracks created BrowserContexts for cleanup. Auto-removes on context close. */
  private contexts = new Set<BrowserContext>();

  /**
   * Ensure the shared browser is available.
   * Uses singleton from `utils/browser.ts` — safe to call multiple times.
   */
  async init(): Promise<void> {
    await getBrowser();
  }

  /**
   * Close all tracked contexts. The shared browser is NOT closed
   * (other scrapers may still be using it). Call `closeBrowser()` on shutdown.
   */
  async close(): Promise<void> {
    for (const ctx of this.contexts) {
      try {
        await ctx.close();
      } catch (err) {
        this.log.warn({ err }, 'Failed to close browser context');
      }
    }
    this.contexts.clear();
  }

  /**
   * Create a new page with stealth configuration.
   *
   * Uses the shared browser singleton. Each call creates a fresh
   * BrowserContext (isolation). The context is tracked for cleanup on
   * {@link close} and auto-removed when the context closes.
   *
   * @param reuseContext - If true, reuses the last created context (useful for
   *   multi-page scrapes where isolation isn't required). Default: false.
   */
  protected async newPage(reuseContext = false): Promise<Page> {
    const browser = await getBrowser();

    let context: BrowserContext;
    if (reuseContext && this.contexts.size > 0) {
      // Get the most recently created context
      context = [...this.contexts].pop()!;
    } else {
      context = await browser.newContext(stealthConfig);

      // Auto-remove from tracking when context closes (prevents unbounded growth)
      context.on('close', () => this.contexts.delete(context));
      this.contexts.add(context);
    }

    const page = await context.newPage();
    await page.addInitScript(STEALTH_SCRIPTS);
    return page;
  }

  /**
   * Public wrapper for creating a new page with stealth.
   * Used by custom modules that need isolated browser contexts.
   */
  async createPage(): Promise<Page> {
    return this.newPage(false);
  }

  /**
   * Send raw content to Ollama for structured job data extraction.
   *
   * @returns Validated {@link JobExtractionResult} from the LLM.
   * @throws {LLMExtractionError} if the LLM output cannot be parsed or validated.
   */
  protected async extractWithLLM(rawContent: string): Promise<JobExtractionResult> {
    // Guard against empty or excessively large input
    if (!rawContent || rawContent.trim().length === 0) {
      throw new Error('extractWithLLM: empty content');
    }
    if (rawContent.length > 200_000) {
      this.log.warn({ length: rawContent.length }, 'Truncating content for LLM');
    }
    return this.ollama.extractJobData(rawContent.slice(0, 200_000));
  }

  /**
   * Public wrapper for LLM-based job extraction.
   * Used by custom modules for structured extraction.
   */
  async extractJobData(rawContent: string): Promise<JobExtractionResult> {
    return this.extractWithLLM(rawContent);
  }

  /**
   * Retry a function with exponential backoff and jitter.
   *
   * @param scrapeFn - The async function to retry.
   * @param retries - Maximum number of attempts (default 3).
   * @param delayMs - Base delay in milliseconds (doubles each attempt).
   * @param maxDelayMs - Maximum delay cap (default 30000ms).
   * @returns The result of `scrapeFn` on success.
   * @throws The last error if all retries fail.
   */
  protected async scrapeWithRetry<T>(
    scrapeFn: () => Promise<T>,
    retries = 3,
    delayMs = 2000,
    maxDelayMs = 30_000,
  ): Promise<T> {
    let lastError: Error | null = null;
    for (let attempt = 0; attempt < retries; attempt++) {
      try {
        return await scrapeFn();
      } catch (error) {
        lastError = error instanceof Error ? error : new Error(String(error));
        if (attempt < retries - 1) {
          const backoff = Math.min(delayMs * 2 ** attempt, maxDelayMs);
          // Symmetric jitter: ±10%
          const jitter = backoff * (0.9 + Math.random() * 0.2);
          await new Promise(resolve => setTimeout(resolve, jitter));
        }
      }
    }
    throw lastError;
  }

  /**
   * Scrape jobs from the given base URL.
   *
   * @param baseUrl - The site's job listing URL.
   * @param keywords - Optional search keywords to filter results.
   * @param location - Optional location filter.
   * @param signal - Optional AbortSignal for cancellation.
   * @returns Array of normalized job postings.
   */
  abstract scrape(
    baseUrl: string,
    keywords?: string[],
    location?: string,
    signal?: AbortSignal
  ): Promise<ScrapedJob[]>;
}

