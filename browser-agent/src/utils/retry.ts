/**
 * Retry an async function with exponential backoff and jitter.
 *
 * @param fn      - Async function to retry. Called up to `maxAttempts` times.
 * @param options - Retry configuration (or just `maxAttempts` as a number).
 * @returns The result of `fn` on first success.
 * @throws The last error from `fn` if all attempts fail.
 * @throws TypeError if `maxAttempts < 1` or `delay < 0`.
 * @throws DOMException('AbortError') if the provided signal is aborted.
 *
 * @example
 *   // Simple — 3 attempts, 1s base delay
 *   const data = await retry(async () => {
 *     const r = await fetch(url);
 *     return r.json();
 *   });
 *
 * @example
 *   // With options and logging
 *   await retry(
 *     () => submitForm(page),
 *     {
 *       maxAttempts: 3,
 *       delay: 2000,
 *       onRetry: (err, attempt) =>
 *         logger.warn({ err, attempt }, 'Form submit retry'),
 *     }
 *   );
 *
 * @example
 *   // Selective retry — skip 4xx client errors
 *   await retry(
 *     () => ollama.generate(prompt),
 *     {
 *       maxAttempts: 3,
 *       delay: 1000,
 *       shouldRetry: (err) => !(err instanceof HttpError && err.status >= 400 && err.status < 500),
 *     }
 *   );
 *
 * @example
 *   // Cancellable retry
 *   const controller = new AbortController();
 *   await retry(
 *     () => scrape(url),
 *     { maxAttempts: 5, signal: controller.signal },
 *   );
 */

export interface RetryOptions {
  /** Maximum number of attempts (default 3). Must be >= 1. */
  maxAttempts?: number;
  /** Base delay in ms between retries. Doubles each attempt. Must be >= 0. */
  delay?: number;
  /** Add jitter to avoid thundering herd. Default true. */
  jitter?: boolean;
  /** AbortSignal to cancel retries. Throws AbortError when aborted. */
  signal?: AbortSignal;
  /** Called before each retry. Receives the error and 1-indexed attempt number. */
  onRetry?: (error: unknown, attempt: number) => void;
  /**
   * Decide whether to retry on this error. Return `false` to fail immediately.
   * Called after each failure to decide whether to retry.
   * Default: always retry (returns `true`).
   */
  shouldRetry?: (error: unknown) => boolean;
}

/**
 * Apply jitter to a delay value: random between 50% and 100% of the base.
 * Prevents multiple callers from retrying in lockstep.
 * @internal
 */
function withJitter(delayMs: number): number {
  return Math.floor(delayMs * (0.5 + Math.random() * 0.5));
}

export async function retry<T>(
  fn: () => Promise<T>,
  options?: RetryOptions | number,
): Promise<T> {
  const opts = typeof options === 'number'
    ? { maxAttempts: options }
    : options ?? {};

  const maxAttempts = opts.maxAttempts ?? 3;
  const delay = opts.delay ?? 1000;
  const jitter = opts.jitter ?? true;

  if (maxAttempts < 1) {
    throw new TypeError(`maxAttempts must be >= 1, got ${maxAttempts}`);
  }
  if (delay < 0) {
    throw new TypeError(`delay must be >= 0, got ${delay}`);
  }

  for (let i = 0; i < maxAttempts; i++) {
    try {
      return await fn();
    } catch (error) {
      // Check shouldRetry before deciding whether to continue
      if (opts.shouldRetry && !opts.shouldRetry(error)) {
        throw error;
      }

      // Last attempt — throw the error
      if (i === maxAttempts - 1) throw error;

      // Check abort signal before sleeping
      if (opts.signal?.aborted) {
        throw new DOMException('Retry aborted', 'AbortError');
      }

      opts.onRetry?.(error, i + 1);

      const base = delay * 2 ** i;
      const sleepMs = jitter ? withJitter(base) : base;

      // Use AbortSignal's built-in timeout support (ES2023) or manual listener
      // that is not race.
      if (opts.signal) {
        // Check if already aborted before waiting
        if (opts.signal.aborted) {
          throw new DOMException('Retry aborted', 'AbortError');
        }
        // Use AbortSignal.any for race between timeout and abort
        const abortPromise = new Promise<never>((_, reject) => {
          opts.signal!.addEventListener('abort', () => {
            reject(new DOMException('Retry aborted', 'AbortError'));
          }, { once: true });
        });

        try {
          await Promise.race([
            new Promise<void>(resolve => setTimeout(resolve, sleepMs)),
            abortPromise,
          ]);
        } catch (e) {
          if (e instanceof DOMException && e.name === 'AbortError') throw e;
          throw e;
        }
      } else {
        await new Promise<void>(resolve => setTimeout(resolve, sleepMs));
      }
    }
  }

  // Unreachable — TypeScript needs this for control flow analysis.
  throw new Error('retry: unexpected fallthrough');
}
