import express, { NextFunction, Request, Response } from 'express';
import { z } from 'zod';
import { fillApplicationForm } from './form-filler/index.js';
import { logger } from './utils/logger.js';
import { initializeScrapers, closeScrapers, selectScraperBySourceId, getAllowedDomains } from './scrapers/registry.js';
import { closeBrowser } from './utils/browser.js';

const app = express();
app.use(express.json({ limit: '10mb' }));

const DEFAULT_PORT = 3000;
const PORT = Number(process.env.PORT) || DEFAULT_PORT;
const SCRAPE_TIMEOUT_MS = 5 * 60 * 1000; // 5 minutes — matches Go worker's job_discovery timeout

// ----- Request/Response Schemas (Zod = single source of truth) -----

const scrapeJobsSchema = z.object({
  source_id: z.string().min(1),
  base_url: z.string().url(),
  keywords: z.array(z.string()),
  location: z.string(),
  config: z.record(z.unknown()).optional(),
});

const fillFormSchema = z.object({
  portal_url: z.string().url(),
  portal_type: z.string().min(1),
  form_data: z.record(z.string()),
  resume_path: z.string().min(1).optional(),
  cover_letter_path: z.string().min(1).optional(),
  portfolio_path: z.string().min(1).optional(),
});

const checkEmailsSchema = z.object({
  tenant_id: z.string().min(1),
  client_id: z.string().min(1),
  client_secret: z.string().min(1),
  folders: z.array(z.string()).min(1),
  application_id: z.string().optional(),
});

// Derived types (always in sync with schemas)
type ScrapeJobsRequest = z.infer<typeof scrapeJobsSchema>;
type FillFormRequest = z.infer<typeof fillFormSchema>;
type CheckEmailsRequest = z.infer<typeof checkEmailsSchema>;

// ----- Error envelope helpers -----

function errorResponse(code: string, message: string, details?: unknown) {
  return { error: { code, message, ...(details !== undefined ? { details } : {}) } };
}

function notImplementedResponse(feature: string) {
  return { error: { code: 'NOT_IMPLEMENTED', message: `${feature} not yet implemented` } };
}

function serviceUnavailableResponse(feature: string) {
  return { error: { code: 'SERVICE_UNAVAILABLE', message: `${feature} temporarily unavailable` } };
}

// ----- SSRF protection -----

/**
 * Validate that a URL's hostname is in the allowed domains list.
 * Prevents SSRF attacks via user-supplied URLs.
 */
function validateAllowedUrl(url: string, allowedDomains: string[], context: string): void {
  try {
    const hostname = new URL(url).hostname;
    const isAllowed = allowedDomains.some(domain => hostname === domain || hostname.endsWith(`.${domain}`));
    if (!isAllowed) {
      logger.warn({ url, hostname, allowedDomains, context }, 'Blocked SSRF attempt');
      throw new Error(`URL not in allowed domains: ${hostname}`);
    }
  } catch (err) {
    if (err instanceof Error && err.message.includes('not in allowed domains')) {
      throw err;
    }
    logger.warn({ url, err, context }, 'Failed to parse URL for SSRF validation');
    throw new Error(`Invalid URL for ${context}`);
  }
}

// ----- Handlers -----

/**
 * Health check endpoint.
 * Returns service status and current time.
 */
app.get('/health', (_req: Request, res: Response) => {
  res.json({ status: 'healthy', service: 'browser-agent', time: new Date().toISOString() });
});

/**
 * Scrape job listings from a configured source.
 * Returns partial results if some scrapers fail (errors[]).
 */
app.post('/api/v1/scrape/jobs', async (req: Request, res: Response, next: NextFunction) => {
  const parsed = scrapeJobsSchema.safeParse(req.body);
  if (!parsed.success) {
    return res.status(400).json(errorResponse('INVALID_REQUEST', 'Invalid request body', parsed.error.issues));
  }
  const payload = parsed.data;

  // Select scraper by source_id from config (returns allowed domains too)
  const result = selectScraperBySourceId(payload.source_id);
  if (!result) {
    return res.status(400).json(errorResponse('UNKNOWN_SOURCE', `No scraper found for source_id: ${payload.source_id}`));
  }
  const { scraper, allowedDomains } = result;

  // SSRF protection: validate base_url against scraper-specific allowed domains
  try {
    validateAllowedUrl(payload.base_url, allowedDomains, 'scrape/jobs');
  } catch (err) {
    return res.status(400).json(errorResponse('SSRF_BLOCKED', err instanceof Error ? err.message : 'URL not allowed'));
  }

  // Use AbortController for proper cancellation
  const controller = new AbortController();
  const timeoutHandle = setTimeout(() => controller.abort(), SCRAPE_TIMEOUT_MS);

  try {
    const jobs = await Promise.race([
      scraper(payload.base_url, payload.keywords, payload.location, controller.signal),
      new Promise<never>((_, reject) =>
        controller.signal.addEventListener('abort', () =>
          reject(new Error(`Scrape timed out after ${SCRAPE_TIMEOUT_MS}ms`))
        )
      ),
    ]);
    return res.json({ jobs, source: 'custom', scrape_errors: [] as string[] });
  } catch (e) {
    return next(e);
  } finally {
    clearTimeout(timeoutHandle);
  }
});

/**
 * Fill and submit an application form.
 */
app.post('/api/v1/forms/fill', async (req: Request, res: Response, next: NextFunction) => {
  const parsed = fillFormSchema.safeParse(req.body);
  if (!parsed.success) {
    return res.status(400).json(errorResponse('INVALID_REQUEST', 'Invalid request body', parsed.error.issues));
  }
  const payload = parsed.data;

  // SSRF protection: validate portal_url against all scraper domains
  const allowedDomains = getAllowedDomains();
  try {
    validateAllowedUrl(payload.portal_url, allowedDomains, 'forms/fill');
  } catch (err) {
    return res.status(400).json(errorResponse('SSRF_BLOCKED', err instanceof Error ? err.message : 'URL not allowed'));
  }

  const result = await fillApplicationForm({
    url: payload.portal_url,
    candidateData: payload.form_data,
    resumePath: payload.resume_path,
    coverLetterPath: payload.cover_letter_path,
    portfolioPath: payload.portfolio_path,
  });

  return res.json({
    success: result.success,
    message: result.errors.length > 0 ? result.errors.join('; ') : 'Form filled successfully',
    filled_fields: result.filledFields,
    errors: result.errors,
  });
});

/**
 * Check for job-related emails via Microsoft Graph.
 * Currently a placeholder — implementation pending.
 */
app.post('/api/v1/emails/check', async (req: Request, res: Response, next: NextFunction) => {
  const parsed = checkEmailsSchema.safeParse(req.body);
  if (!parsed.success) {
    return res.status(400).json(errorResponse('INVALID_REQUEST', 'Invalid request body', parsed.error.issues));
  }
  // TODO: Implement Microsoft Graph email checking
  return res.status(501).json(notImplementedResponse('Email checking'));
});

// ----- Global error middleware (must be last) -----

// eslint-disable-next-line @typescript-eslint/no-unused-vars
app.use((err: Error, req: Request, res: Response, _next: NextFunction) => {
  logger.error(
    { err, path: req.path, method: req.method, query: req.query },
    'unhandled error in browser-agent handler',
  );
  // Do NOT leak internal error details to clients
  if (err.name === 'ZodError') {
    return res.status(400).json(errorResponse('INVALID_REQUEST', 'Validation failed'));
  }
  res.status(500).json(errorResponse('INTERNAL_ERROR', 'Internal server error'));
});

// ----- Server lifecycle -----

let isShuttingDown = false;
let signalsRegistered = false;

async function startServer() {
  await initializeScrapers();

  const server = app.listen(PORT, () => {
    logger.info({ message: 'Browser Agent server started', port: PORT });
  });

  // Graceful shutdown - register signals only once
  if (!signalsRegistered) {
    signalsRegistered = true;
    for (const signal of ['SIGINT', 'SIGTERM'] as const) {
      process.on(signal, async () => {
        if (isShuttingDown) return;
        isShuttingDown = true;

        logger.info({ message: 'Shutdown signal received, closing server', signal });

        server.close(async () => {
          await closeScrapers();
          await closeBrowser();
          logger.info({ message: 'Server closed, browser agent resources released' });
          process.exit(0);
        });

        // Force exit after 10s
        setTimeout(() => {
          logger.error({ message: 'Forced shutdown after timeout' });
          process.exit(1);
        }, 10_000);
      });
    }
  }
}

// Auto-start only when run directly (not imported for testing)
const isMainModule = require.main === module;
if (isMainModule) {
  startServer().catch(err => {
    logger.error({ err }, 'Failed to start server');
    process.exit(1);
  });
}

export { app, startServer };
export type { ScrapeJobsRequest, FillFormRequest, CheckEmailsRequest };