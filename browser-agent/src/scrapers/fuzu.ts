/**
 * Placeholder scraper for Fuzu job board.
 *
 * Tier 2 (hybrid): semi-structured HTML with sometimes embedded JSON.
 * Not yet implemented — will throw on invocation.
 *
 * When implementing:
 *   - Try extracting embedded JSON/`__NEXT_DATA__` first (cheap)
 *   - Fall back to Playwright + LLM extraction
 */
export async function scrapeFuzu(_baseUrl: string, _keywords: string[], _location: string): Promise<never[]> {
  throw new Error('Fuzu scraper not yet implemented');
}
