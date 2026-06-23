/**
 * Tests for custom scraper helpers (scrapers/custom/helpers.ts).
 *
 * Covers: hashId, inferCompany, extractDomain, detectSourceFromUrl.
 * Pure functions — no external dependencies.
 */

import { hashId, inferCompany, extractDomain, detectSourceFromUrl, SOURCE_CUSTOM } from '../helpers';

describe('hashId', () => {
  it('generates deterministic ID from url, title, company', () => {
    const id1 = hashId('https://example.com/job/1', 'Engineer', 'Acme');
    const id2 = hashId('https://example.com/job/1', 'Engineer', 'Acme');
    expect(id1).toBe(id2);
  });

  it('starts with custom- prefix', () => {
    const id = hashId('https://example.com/job', 'Title', 'Company');
    expect(id).toMatch(/^custom-/);
  });

  it('is 36 chars total (custom- + 32 hex)', () => {
    const id = hashId('https://example.com/job', 'Title', 'Company');
    expect(id).toHaveLength(39); // "custom-" (7) + 32 hex = 39
  });

  it('produces different IDs for different inputs', () => {
    const id1 = hashId('https://example.com/job/1', 'Engineer', 'Acme');
    const id2 = hashId('https://example.com/job/2', 'Engineer', 'Acme');
    const id3 = hashId('https://example.com/job/1', 'Designer', 'Acme');
    expect(id1).not.toBe(id2);
    expect(id1).not.toBe(id3);
  });
});

describe('inferCompany', () => {
  it('extracts company from careers subdomain', () => {
    expect(inferCompany('https://careers.example.com/jobs/123')).toBe('example');
  });

  it('extracts company from jobs subdomain', () => {
    expect(inferCompany('https://jobs.stripe.com/role')).toBe('stripe');
  });

  it('extracts company from two-part domain', () => {
    expect(inferCompany('https://acme.com/careers')).toBe('acme');
  });

  it('handles www prefix', () => {
    expect(inferCompany('https://www.example.com/jobs')).toBe('example');
  });

  it('returns ATS name for known ATS domains', () => {
    expect(inferCompany('https://boards.greenhouse.io/acme')).toBe('greenhouse');
    expect(inferCompany('https://jobs.lever.co/acme')).toBe('lever');
  });

  it('returns Unknown for invalid URL', () => {
    expect(inferCompany('not-a-url')).toBe('Unknown');
  });

  it('handles three-part domain without common subdomain', () => {
    expect(inferCompany('https://blog.example.com/jobs')).toBe('example');
  });
});

describe('extractDomain', () => {
  it('extracts origin from URL', () => {
    expect(extractDomain('https://careers.example.com/jobs/123')).toBe('https://careers.example.com');
  });

  it('handles URL without path', () => {
    expect(extractDomain('https://example.com')).toBe('https://example.com');
  });

  it('returns empty string for invalid URL', () => {
    expect(extractDomain('not-a-url')).toBe('');
  });
});

describe('detectSourceFromUrl', () => {
  it('detects greenhouse', () => {
    expect(detectSourceFromUrl('https://boards.greenhouse.io/acme')).toBe('greenhouse');
  });

  it('detects lever', () => {
    expect(detectSourceFromUrl('https://jobs.lever.co/acme')).toBe('lever');
  });

  it('detects indeed', () => {
    expect(detectSourceFromUrl('https://indeed.com/viewjob?jk=123')).toBe('indeed');
  });

  it('detects remoteok', () => {
    expect(detectSourceFromUrl('https://remoteok.com/remote-jobs/123')).toBe('remoteok');
  });

  it('detects ashby', () => {
    expect(detectSourceFromUrl('https://jobs.ashbyhq.com/acme')).toBe('ashby');
  });

  it('returns custom for unknown URL', () => {
    expect(detectSourceFromUrl('https://example.com/jobs')).toBe(SOURCE_CUSTOM);
  });

  it('is case-insensitive', () => {
    expect(detectSourceFromUrl('https://GREENHOUSE.io/jobs')).toBe('greenhouse');
  });
});
