import { ScrapedJob } from '../base.js';
import { logger } from '../../utils/logger.js';

const log = logger.child({ component: 'Deduplicator' });

/**
 * Deduplicate jobs by external_id.
 * Keeps the first occurrence (higher-priority strategy).
 * Jobs with empty external_id are always kept (treated as unique).
 *
 * @param jobs - Array of scraped jobs to deduplicate
 * @returns Deduplicated array preserving original order
 */
export function deduplicate(jobs: ScrapedJob[]): ScrapedJob[] {
  const seen = new Set<string>();
  const unique: ScrapedJob[] = [];
  let duplicates = 0;

  for (const job of jobs) {
    const id = job.external_id?.trim();

    // Treat empty/missing external_id as unique (always keep)
    if (!id) {
      unique.push(job);
      continue;
    }

    if (!seen.has(id)) {
      seen.add(id);
      unique.push(job);
    } else {
      duplicates++;
    }
  }

  if (duplicates > 0) {
    log.debug({ input: jobs.length, output: unique.length, duplicates }, 'Deduplication complete');
  }

  return unique;
}