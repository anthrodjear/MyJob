/**
 * Placeholder scraper for MyJobMag job board.
 *
 * Tier 2/3 (HTML + LLM): no clean API, traditional CMS-rendered pages.
 * Not yet implemented — will throw on invocation.
 *
 * When implementing:
 *   - Playwright for HTML rendering (CMS pages, category pagination)
 *   - LLM extraction for structured job data
 */
export async function scrapeMyJobMag(_baseUrl: string, _keywords: string[], _location: string): Promise<never[]> {
  throw new Error('MyJobMag scraper not yet implemented');
}
