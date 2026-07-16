import { scrapeGreenhouse } from './greenhouse.js';
import { scrapeLever } from './lever.js';
import { scrapeRemoteOK } from './remoteok.js';
import { scrapeIndeed } from './indeed.js';
import { CustomScraper } from './custom/custom-scraper.js';
import { ScrapedJob } from './base.js';
import { logger } from '../utils/logger.js';
import { loadJobSites, JobSiteConfig } from '../config/jobsites.js';

const log = logger.child({ component: 'ScraperRegistry' });

/**
 * Type alias for scraper function signature.
 */
export type ScraperFn = (baseUrl: string, keywords?: string[], location?: string, signal?: AbortSignal) => Promise<ScrapedJob[]>;

/**
 * Registry of all available scrapers.
 * Maps source identifiers to scraper functions/classes.
 */
export interface ScraperEntry {
  name: string;
  scrape: ScraperFn;
  patterns: RegExp[];
  /** Allowed domains for SSRF protection — derived from patterns */
  allowedDomains: string[];
}

/**
 * Map job site type to its scraper function.
 */
const SCRAPER_FACTORY: Record<string, (site: JobSiteConfig) => ScraperFn> = {
  greenhouse: (_site) => (baseUrl, _keywords, _location) => scrapeGreenhouse(baseUrl),
  lever: (_site) => (baseUrl, _keywords, _location) => scrapeLever(baseUrl),
  remoteok: (_site) => (baseUrl, keywords, _location) => scrapeRemoteOK(baseUrl, keywords ?? []),
  indeed: (_site) => (baseUrl, keywords, location) => scrapeIndeed(baseUrl, keywords ?? [], location ?? ''),
};

/**
 * Derive a URL-matching RegExp from a job site's base_url.
 * Converts e.g. "https://boards.greenhouse.io" to /greenhouse\.io/i
 */
function derivePatternFromBaseUrl(baseUrl: string): RegExp {
  try {
    const hostname = new URL(baseUrl).hostname;
    // Escape dots for regex safety
    const escaped = hostname.replace(/\./g, '\\.');
    return new RegExp(escaped, 'i');
  } catch {
    // Fallback: match anything (should never happen with validated config)
    return /./i;
  }
}

/**
 * Build scraper registry from job sites configuration.
 * Fault-tolerant: one bad config doesn't crash all scrapers.
 * Returns an array of ScraperEntry with patterns and domains derived from config.
 */
function buildScraperRegistry(): ScraperEntry[] {
  let jobSites: JobSiteConfig[];
  try {
    jobSites = loadJobSites();
  } catch (err) {
    log.error({ err }, 'Failed to load job sites config, returning empty registry');
    return [];
  }

  const entries: ScraperEntry[] = [];

  for (const site of jobSites) {
    try {
      // Skip custom type - handled by CustomScraper fallback
      if (site.type === 'custom') continue;

      const factory = SCRAPER_FACTORY[site.type];
      if (!factory) {
        log.warn({ type: site.type, name: site.name }, 'Unknown scraper type, skipping');
        continue;
      }

      // Derive pattern from base_url
      const patterns = [derivePatternFromBaseUrl(site.base_url)];

      // Extract allowed domains from base_url and api_url
      const allowedDomains = new Set<string>();
      try {
        allowedDomains.add(new URL(site.base_url).hostname);
      } catch {
        // Ignore invalid URL
      }
      if (site.api_url) {
        try {
          allowedDomains.add(new URL(site.api_url).hostname);
        } catch {
          // Ignore invalid URL
        }
      }

      entries.push({
        name: site.name,
        scrape: factory(site),
        patterns,
        allowedDomains: Array.from(allowedDomains),
      });
    } catch (err) {
      log.warn({ err, name: site.name }, 'Failed to register scraper, skipping');
    }
  }

  return entries;
}

/**
 * Scraper registry — single source of truth for all scraping strategies.
 * Populated dynamically from config/jobsites/*.yaml files.
 * Adding a new scraper = add YAML file in config/jobsites/.
 *
 * Lazily initialized on first access to avoid crashing on import if config is bad.
 */
let _scrapers: ScraperEntry[] | null = null;
export function getScrapers(): ScraperEntry[] {
  if (_scrapers === null) {
    _scrapers = buildScraperRegistry();
    log.info({ count: _scrapers.length }, 'Scraper registry loaded');
  }
  return _scrapers;
}

/**
 * Get all allowed domains from the scraper registry for SSRF protection.
 * Includes dedicated scraper domains and custom fallback domains.
 * Deduplicates loadJobSites() calls by using the same cached data.
 */
export function getAllowedDomains(): string[] {
  const domains = new Set<string>();

  const scrapers = getScrapers();
  for (const entry of scrapers) {
    for (const domain of entry.allowedDomains) {
      domains.add(domain);
    }
  }

  // Add common ATS domains that CustomScraper might encounter
  // These are loaded from the custom sites config (cached by loadJobSites)
  try {
    const jobSites = loadJobSites();
    const customSiteConfig = jobSites.find(s => s.type === 'custom');
    if (customSiteConfig?.sites) {
      for (const customSite of customSiteConfig.sites) {
        try {
          domains.add(new URL(customSite.url).hostname);
        } catch {
          // Ignore invalid URL
        }
      }
    }
  } catch {
    // Config load already logged — continue with what we have
  }

  return Array.from(domains);
}

/**
 * CustomScraper instance — reused across requests.
 * Falls back to generic scraping for unknown sources.
 */
const customScraper = new CustomScraper();

/**
 * Select the appropriate scraper for a given URL.
 * Returns the scraper function and source identifier.
 * Falls back to CustomScraper for unknown sources.
 */
export function selectScraper(url: string): { scraper: ScraperFn; source: string } {
  const scrapers = getScrapers();
  for (const entry of scrapers) {
    for (const pattern of entry.patterns) {
      if (pattern.test(url)) {
        log.debug({ url, source: entry.name, event: 'scraper_selected' }, 'Selected dedicated scraper');
        return { scraper: entry.scrape, source: entry.name };
      }
    }
  }

  // Fallback to CustomScraper
  log.debug({ url, event: 'fallback_selected' }, 'No dedicated scraper matched, using CustomScraper');
  return {
    scraper: (baseUrl, keywords, location) => customScraper.scrape(baseUrl, keywords, location),
    source: 'custom',
  };
}

/**
 * Select scraper by source_id from job sites configuration.
 * Returns the scraper function, source identifier, and allowed domains.
 * Falls back to null if source_id not found.
 */
export function selectScraperBySourceId(sourceId: string): { scraper: ScraperFn; source: string; allowedDomains: string[] } | null {
  const jobSites = loadJobSites();
  const site = jobSites.find(s => s.name === sourceId || s.type === sourceId);
  
  if (!site) {
    return null;
  }

  // Custom type uses CustomScraper
  if (site.type === 'custom') {
    const domains: string[] = [];
    if (site.sites) {
      for (const s of site.sites) {
        try {
          domains.push(new URL(s.url).hostname);
        } catch { /* ignore */ }
      }
    }
    return {
      scraper: (baseUrl, keywords, location) => customScraper.scrape(baseUrl, keywords, location),
      source: 'custom',
      allowedDomains: domains,
    };
  }

  const factory = SCRAPER_FACTORY[site.type];
  if (!factory) {
    return null;
  }

  // Derive allowed domains from config
  const allowedDomains = new Set<string>();
  try {
    allowedDomains.add(new URL(site.base_url).hostname);
  } catch { /* ignore */ }
  if (site.api_url) {
    try {
      allowedDomains.add(new URL(site.api_url).hostname);
    } catch { /* ignore */ }
  }

  return { scraper: factory(site), source: site.name, allowedDomains: Array.from(allowedDomains) };
}

/**
 * Initialize all scrapers (e.g. browser setup for CustomScraper).
 */
export async function initializeScrapers(): Promise<void> {
  try {
    await customScraper.init();
    log.info({ event: 'scrapers_initialized' }, 'All scrapers initialized');
  } catch (err) {
    log.error({ err, event: 'scrapers_init_failed' }, 'Failed to initialize scrapers');
    throw err;
  }
}

/**
 * Close all scrapers (cleanup browser contexts).
 */
export async function closeScrapers(): Promise<void> {
  try {
    await customScraper.close();
    log.info({ event: 'scrapers_closed' }, 'All scrapers closed');
  } catch (err) {
    log.error({ err, event: 'scrapers_close_failed' }, 'Failed to close scrapers');
    // Don't rethrow — cleanup should be best-effort
  }
}