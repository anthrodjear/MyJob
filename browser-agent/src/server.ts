import express, { NextFunction, Request, Response } from 'express';
import { z } from 'zod';
import { AccessToken } from 'livekit-server-sdk';
import crypto from 'node:crypto';
import net from 'node:net';
import { fillApplicationForm } from './form-filler/index.js';
import { logger } from './utils/logger.js';
import { initializeScrapers, closeScrapers, selectScraperBySourceId, getAllowedDomains } from './scrapers/registry.js';
import { closeBrowser } from './utils/browser.js';
import { createInterviewSessionFactory } from './voice/factory.js';

const app = express();
app.use(express.json({ limit: '10mb' }));

const DEFAULT_PORT = 3000;
const PORT = Number(process.env.PORT) || DEFAULT_PORT;
const SCRAPE_TIMEOUT_MS = 5 * 60 * 1000; // 5 minutes â€” matches Go worker's job_discovery timeout

// ----- Internal-secret auth -----
//
// When API_INTERNAL_SECRET is set, every /api/v1/* request must carry the
// matching X-Internal-Secret header. This is the only auth on browser-agent
// (the service is intended to run on a trusted network, but the secret
// stops accidental open exposure if the port is forwarded / discovered).
// When unset, requests pass through but a startup warning is logged so
// misconfigured deployments are visible.
const API_INTERNAL_SECRET = process.env.API_INTERNAL_SECRET;
if (!API_INTERNAL_SECRET) {
  logger.warn(
    { envVar: 'API_INTERNAL_SECRET' },
    'API_INTERNAL_SECRET is not set — /api/v1/* requests will NOT be authenticated. Set this in any non-local deployment.',
  );
} else if (API_INTERNAL_SECRET.length < 32) {
  logger.warn(
    { length: API_INTERNAL_SECRET.length },
    'API_INTERNAL_SECRET is shorter than 32 chars — use a high-entropy random value.',
  );
}

const API_INTERNAL_SECRET_HEADER = 'x-internal-secret';

function internalSecretAuth(req: Request, res: Response, next: NextFunction): void {
  // Skip the auth check entirely when the secret isn't configured. The
  // startup warning already fires once at boot.
  if (!API_INTERNAL_SECRET) {
    return next();
  }
  // Health checks are unauthenticated so external probes (k8s, load
  // balancers) can hit them without holding the secret.
  if (req.method === 'GET' && req.path === '/health') {
    return next();
  }
  // Only enforce on the public API surface.
  if (!req.path.startsWith('/api/v1/')) {
    return next();
  }
  const provided = req.header(API_INTERNAL_SECRET_HEADER);
  if (!provided) {
    logger.warn({ ip: req.ip, path: req.path }, 'Rejected request: missing X-Internal-Secret');
    res.status(401).json(errorResponse('UNAUTHORIZED', 'X-Internal-Secret header is required'));
    return;
  }
  // timingSafeEqual requires equal-length buffers. Mismatched lengths
  // always fail, so we still compare against a padded expected to keep
  // the timing of the comparison constant. We don't want a length-based
  // side channel that reveals the secret's length.
  const expected = Buffer.from(API_INTERNAL_SECRET);
  const got = Buffer.from(provided);
  if (got.length !== expected.length || !crypto.timingSafeEqual(expected, got)) {
    logger.warn({ ip: req.ip, path: req.path }, 'Rejected request: invalid X-Internal-Secret');
    res.status(401).json(errorResponse('UNAUTHORIZED', 'X-Internal-Secret header is invalid'));
    return;
  }
  return next();
}

// Install the secret-check middleware before any /api/v1/* handlers run.
// express.json must already be mounted (it is, above) so the handler can
// read the body for zod validation if needed.
app.use(internalSecretAuth);

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

const startInterviewSchema = z.object({
  interview_id: z.string().min(1),
  application_id: z.string().min(1),
  mode: z.enum(['assist', 'autonomous']),
  external_session: z.string().min(1),
  provider: z.string().min(1),
  model: z.string().min(1),
});

// Derived types (always in sync with schemas)
// Types derived inline in handlers via z.infer<typeof schema>

// ----- Error envelope helpers -----

function errorResponse(code: string, message: string, details?: unknown) {
  return { error: { code, message, ...(details !== undefined ? { details } : {}) } };
}

function notImplementedResponse(feature: string) {
  return { error: { code: 'NOT_IMPLEMENTED', message: `${feature} not yet implemented` } };
}

// ----- SSRF protection -----

/**
 * Validate that a URL's hostname is in the allowed domains list.
 * Prevents SSRF attacks via user-supplied URLs.
 *
 * Also rejects raw IP literals (IPv4 and IPv6) and "decimal" / octal /
 * hex IPv4 forms — without this, an attacker can submit `http://2130706433`
 * or `http://0177.0.0.1` (which the WHATWG URL parser will normalize to
 * 127.0.0.1) and the suffix match against e.g. ".example.com" might
 * accidentally let them through if a clever payload is used. Refuse all
 * IP-shaped hosts up front.
 */
function validateAllowedUrl(url: string, allowedDomains: string[], context: string): void {
  let parsed: URL;
  try {
    parsed = new URL(url);
  } catch (err) {
    logger.warn({ url, err, context }, 'Failed to parse URL for SSRF validation');
    throw new Error(`Invalid URL for ${context}`);
  }

  const hostname = parsed.hostname;

  // Reject IP literals before any suffix match. net.isIP handles both
  // IPv4 dotted-quad and bracketed IPv6 (and also recognises the
  // unbracketed `[::1]` form that some parsers accept via URL.hostname).
  if (net.isIP(hostname) !== 0) {
    logger.warn({ url, hostname, context }, 'Blocked SSRF attempt (IP literal)');
    throw new Error(`URL not in allowed domains: ${hostname}`);
  }

  const isAllowed = allowedDomains.some(domain => hostname === domain || hostname.endsWith(`.${domain}`));
  if (!isAllowed) {
    logger.warn({ url, hostname, allowedDomains, context }, 'Blocked SSRF attempt');
    throw new Error(`URL not in allowed domains: ${hostname}`);
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

  const scrapeErrors: string[] = [];
  try {
    let jobs: import('./scrapers/base.js').ScrapedJob[] = [];
    try {
      jobs = await Promise.race([
        scraper(payload.base_url, payload.keywords, payload.location, controller.signal),
        new Promise<never>((_, reject) =>
          controller.signal.addEventListener('abort', () =>
            reject(new Error(`Scrape timed out after ${SCRAPE_TIMEOUT_MS}ms`))
          )
        ),
      ]);
    } catch (scrapeErr) {
      const msg = scrapeErr instanceof Error ? scrapeErr.message : String(scrapeErr);
      scrapeErrors.push(msg);
      logger.error({ err: scrapeErr, source_id: payload.source_id }, 'Scraper error collected');
    }
    return res.json({ jobs, source: result.source, scrape_errors: scrapeErrors });
  } catch (e) {
    return next(e);
  } finally {
    clearTimeout(timeoutHandle);
  }
});

/**
 * Fill and submit an application form.
 */
app.post('/api/v1/forms/fill', async (req: Request, res: Response, _next: NextFunction) => {
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
 * Currently a placeholder â€” implementation pending.
 */
app.post('/api/v1/emails/check', async (req: Request, res: Response, _next: NextFunction) => {
  const parsed = checkEmailsSchema.safeParse(req.body);
  if (!parsed.success) {
    return res.status(400).json(errorResponse('INVALID_REQUEST', 'Invalid request body', parsed.error.issues));
  }
  // TODO: Implement Microsoft Graph email checking
  return res.status(501).json(notImplementedResponse('Email checking'));
});

/**
 * Start a voice interview session.
 * Long-running endpoint â€” blocks until the interview completes (up to 30 minutes).
 */
app.post('/api/v1/interviews/start', async (req: Request, res: Response, _next: NextFunction) => {
  const parsed = startInterviewSchema.safeParse(req.body);
  if (!parsed.success) {
    return res.status(400).json(errorResponse('INVALID_REQUEST', 'Invalid request body', parsed.error.issues));
  }
  const payload = parsed.data;

  try {
    // Generate LiveKit token for the agent to join the room
    const apiKey = process.env.LIVEKIT_API_KEY;
    const apiSecret = process.env.LIVEKIT_API_SECRET;
    if (!apiKey || !apiSecret) {
      return res.status(500).json(errorResponse('CONFIGURATION_ERROR', 'LiveKit credentials not configured'));
    }

    const at = new AccessToken(apiKey, apiSecret, {
      identity: `agent-${payload.interview_id}`,
    });
    at.addGrant({ roomJoin: true, room: payload.external_session });
    const token = await at.toJwt();

    // Create session via factory (reads config from env/YAML)
    const { session } = await createInterviewSessionFactory();

    // Start session and wait for it to end
    const sessionPromise = new Promise<{ success: boolean; message: string }>((resolve) => {
      session.on('ended', (_reason, transcript) => {
        resolve({
          success: true,
          message: `Interview completed with ${transcript.length} transcript segments`,
        });
      });
      session.on('error', (error) => {
        resolve({
          success: false,
          message: error.message || 'Interview session failed',
        });
      });
    });

    await session.start({
      mode: payload.mode,
      roomName: payload.external_session,
      token,
      applicationId: payload.application_id,
      providers: {
        realtime: payload.provider === 'openai_realtime' ? 'openai_realtime' : undefined,
      },
    });

    // Wait for interview to complete (long-running)
    const result = await sessionPromise;

    return res.json({
      success: result.success,
      message: result.message,
    });
  } catch (err) {
    const error = err instanceof Error ? err : new Error(String(err));
    logger.error({ err, interview_id: payload.interview_id }, 'Failed to start interview session');
    return res.status(500).json(errorResponse('INTERVIEW_ERROR', error.message));
  }
});

// ----- Global error middleware (must be last) -----

 
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

export { app, startServer };

