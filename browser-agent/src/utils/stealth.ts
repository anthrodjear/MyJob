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
 *   STEALTH_USER_AGENT       — custom User-Agent string
 *   STEALTH_TIMEZONE         — IANA timezone (default: America/New_York)
 *   STEALTH_LOCALE           — BCP-47 locale   (default: en-US)
 *   STEALTH_LATITUDE         — geolocation lat  (default: 40.7128)
 *   STEALTH_LONGITUDE        — geolocation lon  (default: -74.0060)
 *   STEALTH_VIEWPORT_WIDTH   — viewport width  (default: 1920)
 *   STEALTH_VIEWPORT_HEIGHT  — viewport height (default: 1080)
 *   STEALTH_DEVICE_SCALE     — device scale factor (default: 1.5)
 *   STEALTH_COLOR_SCHEME     — 'light' | 'dark' | 'no-preference' (default: light)
 *
 * @example
 *   import { stealthConfig } from '../utils/stealth.js';
 *   const context = await browser.newContext(stealthConfig);
 */
function parseNumberEnv(name: string, fallback: number): number {
  const raw = process.env[name];
  if (raw === undefined) return fallback;
  const parsed = Number(raw);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function parseStringEnv(name: string, fallback: string): string {
  return process.env[name] ?? fallback;
}

export const stealthConfig: Readonly<BrowserContextOptions> = {
  userAgent:
    process.env.STEALTH_USER_AGENT ??
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36',

  viewport: {
    width: parseNumberEnv('STEALTH_VIEWPORT_WIDTH', 1920),
    height: parseNumberEnv('STEALTH_VIEWPORT_HEIGHT', 1080),
  },

  locale: parseStringEnv('STEALTH_LOCALE', 'en-US'),
  timezoneId: parseStringEnv('STEALTH_TIMEZONE', 'America/New_York'),

  deviceScaleFactor: parseNumberEnv('STEALTH_DEVICE_SCALE', 1.5),
  colorScheme: (parseStringEnv('STEALTH_COLOR_SCHEME', 'light') as 'light' | 'dark' | 'no-preference'),

  isMobile: false,
  hasTouch: false,
  reducedMotion: 'no-preference',
  forcedColors: 'none',

  screen: {
    width: parseNumberEnv('STEALTH_SCREEN_WIDTH', 1920),
    height: parseNumberEnv('STEALTH_SCREEN_HEIGHT', 1080),
  },

  permissions: ['geolocation', 'notifications'],
  geolocation: {
    latitude: parseNumberEnv('STEALTH_LATITUDE', 40.7128),
    longitude: parseNumberEnv('STEALTH_LONGITUDE', -74.0060),
  },

  extraHTTPHeaders: {
    'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
    'Accept-Language': `${parseStringEnv('STEALTH_LOCALE', 'en-US')},en;q=0.9`,
    'Accept-Encoding': 'gzip, deflate, br, zstd',
    'Upgrade-Insecure-Requests': '1',
    'Sec-Fetch-Dest': 'document',
    'Sec-Fetch-Mode': 'navigate',
    'Sec-Fetch-Site': 'none',
    'Sec-Fetch-User': '?1',
    'Sec-Ch-Ua': '"Chromium";v="137", "Not=A?Brand";v="24", "Google Chrome";v="137"',
    'Sec-Ch-Ua-Mobile': '?0',
    'Sec-Ch-Ua-Platform': '"Windows"',
  },
};