import { Page } from 'playwright';
import { ScrapedJob } from '../base.js';
import crypto from 'node:crypto';
import { hashId, inferCompany, extractDomain, detectSourceFromUrl } from './helpers.js';
import { logger } from '../../utils/logger.js';

const log = logger.child({ component: 'JsonLdExtractor' });

/**
 * Minimum number of JobPosting objects to trust as a complete listing.
 * Below this threshold, we continue with link discovery to avoid
 * returning only featured/single jobs from JSON-LD.
 */
export const JSONLD_MIN_THRESHOLD = 5;

/** Maximum recursion depth for findJobPostings to prevent stack overflow. */
const MAX_RECURSION_DEPTH = 10;

/**
 * Extract jobs from JSON-LD structured data (schema.org/JobPosting).
 * Many career sites embed this — cheapest extraction path.
 */
export async function extractJsonLd(page: Page, baseUrl: string): Promise<ScrapedJob[]> {
  const results: ScrapedJob[] = [];

  try {
    const jsonLdScripts = await page.locator('script[type="application/ld+json"]').allTextContents();

    log.debug({ baseUrl, scriptsFound: jsonLdScripts.length }, 'Found JSON-LD scripts');

    for (const script of jsonLdScripts) {
      try {
        const data = JSON.parse(script);
        const postings = findJobPostings(data, 0);
        results.push(...postings.map(p => mapJsonLdPosting(p, baseUrl)));
      } catch (err) {
        // Invalid JSON — log and skip
        log.debug({ err, scriptLength: script.length }, 'Failed to parse JSON-LD script');
      }
    }
  } catch (err) {
    // Only catch expected locator/evaluation errors; let unexpected ones propagate
    const errorMessage = err instanceof Error ? err.message : String(err);
    if (errorMessage.includes('Timeout') || errorMessage.includes('Execution context')) {
      log.debug({ baseUrl, err: errorMessage }, 'JSON-LD locator evaluation failed');
      return results;
    }
    throw err;
  }

  log.debug({ baseUrl, jobsExtracted: results.length }, 'JSON-LD extraction complete');
  return results;
}

/**
 * Recursively find JobPosting objects in JSON-LD data.
 * @param depth Current recursion depth, used to prevent stack overflow
 */
function findJobPostings(data: unknown, depth: number): Array<Record<string, unknown>> {
  if (depth > MAX_RECURSION_DEPTH) {
    log.warn({ depth }, 'JSON-LD recursion depth exceeded, stopping');
    return [];
  }

  if (data === null || typeof data !== 'object') return [];

  if (Array.isArray(data)) {
    return data.flatMap(item => findJobPostings(item, depth + 1));
  }

  const obj = data as Record<string, unknown>;

  // Handle @type as string or array
  const typeValue = obj['@type'];
  const isJobPosting = typeValue === 'JobPosting' || (Array.isArray(typeValue) && typeValue.includes('JobPosting'));
  if (isJobPosting) return [obj];

  // Handle ItemList: items are ListItem objects with .item property containing the JobPosting
  if (obj['@type'] === 'ItemList' && Array.isArray(obj.itemListElement)) {
    return obj.itemListElement.flatMap((item: unknown) => {
      if (item && typeof item === 'object') {
        const listItem = item as Record<string, unknown>;
        // Unwrap ListItem.item if present, otherwise recurse into the item itself
        const target = listItem.item ?? item;
        return findJobPostings(target, depth + 1);
      }
      return findJobPostings(item, depth + 1);
    });
  }

  // Handle @graph as array or single object
  if (obj['@graph']) {
    const graph = obj['@graph'];
    if (Array.isArray(graph)) {
      return graph.flatMap((item: unknown) => findJobPostings(item, depth + 1));
    }
    return findJobPostings(graph, depth + 1);
  }

  return [];
}

/**
 * Map a JSON-LD JobPosting to ScrapedJob.
 */
function mapJsonLdPosting(posting: Record<string, unknown>, baseUrl: string): ScrapedJob {
  const org = posting.hiringOrganization as Record<string, unknown> | undefined;
  const loc = posting.jobLocation as Record<string, unknown> | undefined;
  const addr = loc?.address as Record<string, unknown> | undefined;

  // Normalize and validate the listing URL
  const listingUrlRaw = String(posting.url ?? baseUrl);
  let listingUrl: string;
  try {
    listingUrl = new URL(listingUrlRaw, baseUrl).toString();
  } catch {
    listingUrl = baseUrl;
  }

  // Extract salary from baseSalary (MonetaryAmount) or legacy salaryCurrency/salaryValue
  const { salaryMin, salaryMax, salaryCurrency } = extractSalary(posting);

  // Use applicationContact.url if present, otherwise try to find apply link in DOM context,
  // otherwise fall back to listingUrl (for hosted ATS like Greenhouse where listing = application)
  const applicationUrl = extractApplicationUrl(posting, listingUrl);

  // Determine source from the listing URL for consistency with page-level scraping
  const source = detectSourceFromUrl(listingUrl);

  return {
    external_id: hashId(listingUrl, String(posting.title ?? ''), String(org?.name ?? inferCompany(baseUrl))),
    title: String(posting.title ?? 'Unknown Title'),
    company: String(org?.name ?? inferCompany(baseUrl)),
    location: addr ? String(addr.addressLocality ?? addr.addressRegion ?? 'Unknown') : 'Unknown',
    remote_type: inferRemoteTypeFromJsonLd(posting),
    salary_min: salaryMin,
    salary_max: salaryMax,
    salary_currency: salaryCurrency,
    description: String(posting.description ?? ''),
    requirements: '',
    url: listingUrl,
    application_url: applicationUrl,
    company_url: String(org?.url ?? org?.sameAs ?? extractDomain(baseUrl)),
    source,
  };
}

function extractSalary(posting: Record<string, unknown>): { salaryMin: number; salaryMax: number; salaryCurrency: string } {
  const baseSalary = posting.baseSalary as Record<string, unknown> | undefined;
  let salaryMin = 0;
  let salaryMax = 0;
  let salaryCurrency = 'USD';

  if (baseSalary) {
    // MonetaryAmount: baseSalary.value.value (number) and baseSalary.value.currency (string)
    const value = baseSalary.value as Record<string, unknown> | undefined;
    const currency = baseSalary.currency as string | undefined;

    if (value && typeof value.value === 'number') {
      salaryMin = Math.round(value.value);
      salaryMax = Math.round(value.value);
    } else if (value && typeof value.minValue === 'number' && typeof value.maxValue === 'number') {
      salaryMin = Math.round(value.minValue);
      salaryMax = Math.round(value.maxValue);
    }

    if (currency) {
      salaryCurrency = currency;
    }
  }

  // Fallback to legacy salaryCurrency/salaryValue
  if (salaryMin === 0 && salaryMax === 0) {
    const legacyCurrency = posting.salaryCurrency as string | undefined;
    const legacyValue = posting.salaryValue as number | undefined;
    if (legacyValue && typeof legacyValue === 'number') {
      salaryMin = Math.round(legacyValue);
      salaryMax = Math.round(legacyValue);
    }
    if (legacyCurrency) {
      salaryCurrency = legacyCurrency;
    }
  }

  return { salaryMin, salaryMax, salaryCurrency };
}

function extractApplicationUrl(posting: Record<string, unknown>, fallbackUrl: string): string {
  // Check for applicationContact (schema.org JobPosting field)
  const applicationContact = posting.applicationContact as Record<string, unknown> | undefined;
  if (applicationContact && applicationContact.url) {
    try {
      return new URL(String(applicationContact.url), fallbackUrl).toString();
    } catch {
      // Invalid URL, fall through
    }
  }

  // No structured application contact — fall back to listing URL
  // For Greenhouse/Lever-hosted pages, the listing URL IS the application URL
  return fallbackUrl;
}

function inferRemoteTypeFromJsonLd(posting: Record<string, unknown>): string {
  // Check jobLocationType (can be string or array per schema.org)
  const locationType = posting.jobLocationType;
  const types: string[] = Array.isArray(locationType)
    ? locationType.map(t => String(t).toLowerCase())
    : [String(locationType ?? '').toLowerCase()];

  for (const t of types) {
    if (t.includes('remote') || t.includes('telecommute')) return 'remote';
    if (t.includes('hybrid')) return 'hybrid';
    if (t.includes('onsite') || t.includes('on-site')) return 'onsite';
  }

  // Secondary signal: scan description for remote/hybrid/onsite keywords
  const description = String(posting.description ?? '').toLowerCase();
  if (description.includes('remote') || description.includes('work from home') || description.includes('wfh')) {
    return 'remote';
  }
  if (description.includes('hybrid')) return 'hybrid';
  if (description.includes('on-site') || description.includes('onsite') || description.includes('office')) {
    return 'onsite';
  }

  return 'unknown';
}