import crypto from 'node:crypto';

/** Fallback source identifier when no known ATS is detected. */
export const SOURCE_CUSTOM = 'custom';

/** Known ATS domain patterns for source detection. */
const ATS_DOMAIN_PATTERNS: Array<{ pattern: RegExp; source: string }> = [
  { pattern: /greenhouse\.io/i, source: 'greenhouse' },
  { pattern: /lever\.co/i, source: 'lever' },
  { pattern: /ashbyhq\.com/i, source: 'ashby' },
  { pattern: /workable\.com/i, source: 'workable' },
  { pattern: /smartrecruiters\.com/i, source: 'smartrecruiters' },
  { pattern: /workday\.com/i, source: 'workday' },
  { pattern: /icims\.com/i, source: 'icims' },
  { pattern: /bamboohr\.com/i, source: 'bamboohr' },
  { pattern: /jobvite\.com/i, source: 'jobvite' },
  { pattern: /brassring\.com/i, source: 'brassring' },
  { pattern: /taleo\.net/i, source: 'taleo' },
  { pattern: /successfactors\.com/i, source: 'successfactors' },
  { pattern: /oraclecloud\.com/i, source: 'oracle' },
  { pattern: /sap\.com/i, source: 'sap' },
  { pattern: /eightfold\.ai/i, source: 'eightfold' },
  { pattern: /phenompeople\.com/i, source: 'phenom' },
  { pattern: /hirevue\.com/i, source: 'hirevue' },
  { pattern: /modernhire\.com/i, source: 'modernhire' },
  { pattern: /myjobmag/i, source: 'myjobmag' },
  { pattern: /fuzu\.com/i, source: 'fuzu' },
  { pattern: /remoteok\.com/i, source: 'remoteok' },
  { pattern: /indeed\.com/i, source: 'indeed' },
];

/**
 * Generate a deterministic external_id from job attributes.
 * Composite key prevents collisions between different jobs at the same URL.
 * Uses 32 hex chars (128 bits) of SHA-256 for negligible collision probability.
 *
 * @param url - The job listing URL
 * @param title - Job title
 * @param company - Company name
 * @returns Deterministic external ID string
 */
export function hashId(url: string, title: string, company: string): string {
  const composite = `${url}|${title}|${company}`;
  return `custom-${crypto.createHash('sha256').update(composite).digest('hex').slice(0, 32)}`;
}

/**
 * Infer company name from the URL hostname.
 * Attempts to extract the organization name from common career subdomain patterns.
 * e.g. `careers.example.com` → `example`, `jobs.stripe.com` → `stripe`,
 * `example.com` → `example`, `boards.greenhouse.io` → `greenhouse`
 *
 * @param url - The URL to infer company from
 * @returns Inferred company name or 'Unknown'
 */
export function inferCompany(url: string): string {
  try {
    const host = new URL(url).hostname;

    // Handle known ATS hosting patterns: boards.greenhouse.io → greenhouse
    const atsMatch = ATS_DOMAIN_PATTERNS.find(p => p.pattern.test(host));
    if (atsMatch) return atsMatch.source;

    // Remove www. prefix
    const cleanHost = host.replace(/^www\./, '');

    // Split by dots
    const parts = cleanHost.split('.');

    // Common patterns:
    // - careers.company.com → company (3 parts, first is subdomain)
    // - company.com → company (2 parts)
    // - jobs.company.com → company (3 parts)
    // - company.io → company (2 parts)
    if (parts.length >= 3) {
      // Check if first part is a common career subdomain
      const commonSubdomains = ['careers', 'jobs', 'join', 'work', 'team', 'talent', 'hiring', 'recruiting'];
      if (commonSubdomains.includes(parts[0].toLowerCase())) {
        return parts[1];
      }
      // Otherwise second-level domain is likely the company
      return parts[1];
    }

    // 2-part domain: company.tld
    return parts[0];
  } catch {
    return 'Unknown';
  }
}

/**
 * Extract the origin (scheme + host) from a URL.
 * e.g. `https://careers.example.com/jobs/123` → `https://careers.example.com`
 *
 * @param url - The URL to extract domain from
 * @returns Origin string or empty string if invalid
 */
export function extractDomain(url: string): string {
  try {
    return new URL(url).origin;
  } catch {
    return '';
  }
}

/**
 * Detect the ATS source from a URL.
 * Used to infer which form-filling strategy to use downstream.
 *
 * @param url - The URL to check
 * @returns ATS source identifier or 'custom'
 */
export function detectSourceFromUrl(url: string): string {
  for (const { pattern, source } of ATS_DOMAIN_PATTERNS) {
    if (pattern.test(url)) return source;
  }
  return SOURCE_CUSTOM;
}