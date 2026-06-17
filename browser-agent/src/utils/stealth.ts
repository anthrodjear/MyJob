import type { BrowserContextOptions } from 'playwright';

/**
 * Centralised browser fingerprint settings for Playwright stealth.
 *
 * Typed against Playwright's `BrowserContextOptions` for compile-time
 * validation. Import this in any module that creates browser contexts
 * to ensure consistent fingerprinting across all scrapers and form fillers.
 *
 * Environment variables override defaults so the fingerprint can match the
 * runner's actual location, avoiding IP/timezone mismatches that trigger
 * anti-bot detection:
 *
 *   STEALTH_USER_AGENT  — custom User-Agent string
 *   STEALTH_TIMEZONE    — IANA timezone (default: America/New_York)
 *   STEALTH_LOCALE      — BCP-47 locale   (default: en-US)
 *   STEALTH_LATITUDE    — geolocation lat  (default: 40.7128)
 *   STEALTH_LONGITUDE   — geolocation lon  (default: -74.0060)
 *
 * @example
 *   import { stealthConfig } from '../utils/stealth.js';
 *   const context = await browser.newContext(stealthConfig);
 */
export const stealthConfig: Readonly<BrowserContextOptions> = {
  userAgent:
    process.env.STEALTH_USER_AGENT
    ?? 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36',

  viewport: {
    width: 1366,
    height: 768,
  },

  locale: process.env.STEALTH_LOCALE ?? 'en-US',
  timezoneId: process.env.STEALTH_TIMEZONE ?? 'America/New_York',

  deviceScaleFactor: 1.25,
  colorScheme: 'dark',

  permissions: ['geolocation'],
  geolocation: {
    latitude: Number(process.env.STEALTH_LATITUDE) || 40.7128,
    longitude: Number(process.env.STEALTH_LONGITUDE) || -74.0060,
  },

  extraHTTPHeaders: {
    'Accept-Language': `${process.env.STEALTH_LOCALE ?? 'en-US'},en;q=0.9`,
  },
};
