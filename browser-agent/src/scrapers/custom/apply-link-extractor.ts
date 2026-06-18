import { Page } from 'playwright';
import { logger } from '../../utils/logger.js';

const log = logger.child({ component: 'ApplyLinkExtractor' });

/**
 * Selectors for finding apply buttons/links on job detail pages.
 * Ordered from most specific to most general.
 * Uses case-insensitive text matching where possible.
 */
const APPLY_SELECTORS = [
  // Explicit apply link text (case-insensitive via :has-text with regex)
  'a:has-text("(?i)apply now")',
  'a:has-text("(?i)apply on company site")',
  'a:has-text("(?i)submit application")',
  'a:has-text("(?i)start application")',
  'a:has-text("(?i)apply for this job")',
  'a:has-text("(?i)apply online")',
  'a:has-text("(?i)i want to apply")',
  // Common class/id patterns
  'a.apply-button',
  'a#apply-button',
  'a[data-testid="apply-button"]',
  'a[data-apply="true"]',
  'a[data-qa="apply-button"]',
  // href pattern matches
  'a[href*="/apply"]',
  'a[href*="/application"]',
  // Form action handling (some sites use <form action="/apply/123">)
  'form[action*="/apply"]',
  'form[action*="/application"]',
  // Generic fallback (case-insensitive)
  'a:has-text("(?i)apply")',
];

/**
 * Extract the apply URL from a job detail page.
 * Finds apply buttons/links, returns the resolved href.
 * Handles absolute, protocol-relative, and relative URLs.
 */
export async function extractApplyUrl(page: Page): Promise<string | null> {
  for (const selector of APPLY_SELECTORS) {
    try {
      const locator = page.locator(selector).first();

      // Use Playwright's default auto-waiting (no explicit timeout)
      // Locator will wait for element to be attached and visible
      const href = await locator.getAttribute('href');

      // For form actions, check action attribute
      if (!href) {
        const action = await locator.getAttribute('action');
        if (action) {
          log.debug({ selector, action }, 'Found form action attribute');
          return resolveUrl(action, page);
        }
      }

      if (href) {
        log.debug({ selector, href }, 'Found apply link');
        return resolveUrl(href, page);
      }
    } catch (err) {
      log.debug({ selector, err }, 'Selector failed, trying next');
      // Try next selector
    }
  }

  log.debug({ pageUrl: page.url() }, 'No apply link found');
  return null;
}

/**
 * Resolve a URL (absolute, protocol-relative, or relative) against the page URL.
 */
function resolveUrl(href: string, page: Page): string {
  const pageUrl = page.url();

  // Protocol-relative: //example.com → https://example.com
  if (href.startsWith('//')) {
    return `https:${href}`;
  }

  // Absolute: http://... or https://...
  if (href.startsWith('http://') || href.startsWith('https://')) {
    return href;
  }

  // Relative: /apply/123, ./apply, ../careers/apply
  try {
    return new URL(href, pageUrl).toString();
  } catch (err) {
    log.warn({ href, pageUrl, err }, 'Failed to resolve URL');
    return href; // Return as-is as last resort
  }
}