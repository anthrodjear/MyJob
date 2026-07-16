import * as fs from 'node:fs';
import * as path from 'node:path';
import { load, YAMLException, FAILSAFE_SCHEMA } from 'js-yaml';
import { z } from 'zod';

// ----- Zod Schemas -----

const RateLimitSchema = z.object({
  requests_per_second: z.number().positive(),
  daily_cap: z.number().int().positive(),
  respect_robots_txt: z.boolean().default(true),
});

const SelectorsSchema = z.object({
  job_list: z.string().min(1),
  title: z.string().min(1),
  company: z.string().min(1),
  location: z.string().min(1).optional(),
  snippet: z.string().min(1).optional(),
  link: z.string().min(1),
}).passthrough();

const PaginationSchema = z.object({
  type: z.enum(['offset', 'page', 'cursor']),
  param: z.string().min(1),
  per_page: z.number().int().positive().optional(),
  max_pages: z.number().int().positive().optional(),
  cursor_param: z.string().optional(),
}).passthrough();

const CustomSiteSchema = z.object({
  name: z.string().min(1),
  url: z.string().url(),
  selectors: SelectorsSchema.optional(),
  pagination: PaginationSchema.optional(),
});

const BaseJobSiteSchema = z.object({
  name: z.string().min(1),
  type: z.string().min(1),
  base_url: z.string().url(),
  rate_limit: RateLimitSchema,
  api_url: z.string().url().optional(),
  search_url: z.string().min(1).optional(),
  selectors: SelectorsSchema.optional(),
  pagination: PaginationSchema.optional(),
  sites: z.array(CustomSiteSchema).optional(),
});

// ----- Type exports (derived from Zod schemas) -----

export type RateLimitConfig = z.infer<typeof RateLimitSchema>;
export type SelectorsConfig = z.infer<typeof SelectorsSchema>;
export type PaginationConfig = z.infer<typeof PaginationSchema>;
export type CustomSiteConfig = z.infer<typeof CustomSiteSchema>;
export type JobSiteConfig = z.infer<typeof BaseJobSiteSchema>;

export type JobSiteType = JobSiteConfig['type'];

// ----- Custom error type -----

export class JobSitesConfigError extends Error {
  constructor(message: string, public readonly cause?: unknown) {
    super(message);
    this.name = 'JobSitesConfigError';
  }
}

// ----- Loader -----

let jobSitesCache: JobSiteConfig[] | null = null;

/**
 * Load and validate all job site configurations from the config/jobsites/ directory.
 * Returns an array of validated job site configs.
 * Throws JobSitesConfigError on file read errors, YAML parse errors, or schema validation failures.
 */
export function loadJobSites(configDir?: string): JobSiteConfig[] {
  if (jobSitesCache) {
    return jobSitesCache;
  }

  const directoryPath = configDir || path.resolve(process.cwd(), 'config', 'jobsites');

  // 1. Read directory and filter .yaml files
  let files: string[];
  try {
    files = fs.readdirSync(directoryPath)
      .filter(file => file.endsWith('.yaml') || file.endsWith('.yml'))
      .map(file => path.join(directoryPath, file));
  } catch (e) {
    const code = (e as NodeJS.ErrnoException).code;
    if (code === 'ENOENT') {
      throw new JobSitesConfigError(`Job sites config directory not found: ${directoryPath}`, e);
    }
    throw new JobSitesConfigError(`Failed to read job sites config directory: ${directoryPath}`, e);
  }

  if (files.length === 0) {
    throw new JobSitesConfigError(`No YAML config files found in: ${directoryPath}`);
  }

  // 2. Parse and validate each file
  const jobSites: JobSiteConfig[] = [];

  for (const filePath of files) {
    let fileContents: string;
    try {
      fileContents = fs.readFileSync(filePath, 'utf8');
    } catch (e) {
      const code = (e as NodeJS.ErrnoException).code;
      if (code === 'ENOENT') {
        throw new JobSitesConfigError(`Config file not found: ${filePath}`, e);
      }
      throw new JobSitesConfigError(`Failed to read config file: ${filePath}`, e);
    }

    let parsed: unknown;
    try {
      parsed = load(fileContents, {
        filename: path.basename(filePath),
        schema: FAILSAFE_SCHEMA,
      });
    } catch (e) {
      if (e instanceof YAMLException) {
        const line = e.mark?.line !== undefined ? e.mark.line + 1 : '?';
        const col = e.mark?.column !== undefined ? e.mark.column + 1 : '?';
        throw new JobSitesConfigError(
          `Invalid YAML in ${path.basename(filePath)}: ${e.reason} at line ${line}, col ${col}`,
          e,
        );
      }
      throw new JobSitesConfigError('Unknown YAML parsing error', e);
    }

    const result = BaseJobSiteSchema.safeParse(parsed);
    if (!result.success) {
      const issues = result.error.issues
        .map(i => `  - ${i.path.join('.')}: ${i.message}`)
        .join('\n');
      throw new JobSitesConfigError(`Config validation failed for ${path.basename(filePath)}:\n${issues}`, result.error);
    }

    jobSites.push(result.data);
  }

  jobSitesCache = jobSites;
  return jobSites;
}

/**
 * Clear the job sites config cache. Useful for tests that need to reload config.
 */
export function clearJobSitesCache(): void {
  jobSitesCache = null;
}

/**
 * Get a job site config by name.
 * Returns undefined if not found.
 */
export function getJobSiteByName(name: string): JobSiteConfig | undefined {
  const sites = loadJobSites();
  return sites.find(site => site.name === name);
}

/**
 * Get all job site configs of a specific type.
 */
export function getJobSitesByType(type: JobSiteType): JobSiteConfig[] {
  const sites = loadJobSites();
  return sites.filter(site => site.type === type);
}

/**
 * Get all custom site configurations from the custom job sites.
 */
export function getCustomSites(): CustomSiteConfig[] {
  const sites = loadJobSites();
  const customSite = sites.find(s => s.type === 'custom');
  return customSite?.sites ?? [];
}

