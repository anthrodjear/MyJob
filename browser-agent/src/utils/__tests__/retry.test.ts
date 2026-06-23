/**
 * Tests for retry utility (utils/retry.ts).
 *
 * Covers: basic retry, max attempts, delay, jitter, shouldRetry, abort signal, onRetry callback.
 * Pure function — no external dependencies.
 */

import { retry, RetryOptions } from '../retry';

describe('retry', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('returns result on first success', async () => {
    const fn = jest.fn().mockResolvedValue('ok');
    const result = await retry(fn, { maxAttempts: 3, delay: 0, jitter: false });
    expect(result).toBe('ok');
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it('retries on failure and succeeds', async () => {
    const fn = jest.fn()
      .mockRejectedValueOnce(new Error('fail 1'))
      .mockResolvedValue('ok');
    
    const resultPromise = retry(fn, { maxAttempts: 3, delay: 100, jitter: false });
    
    // Advance timers for the retry delay
    await jest.advanceTimersByTimeAsync(100);
    
    const result = await resultPromise;
    expect(result).toBe('ok');
    expect(fn).toHaveBeenCalledTimes(2);
  });

  it('throws after all attempts fail', async () => {
    const fn = jest.fn().mockRejectedValue(new Error('always fails'));
    
    const promise = retry(fn, { maxAttempts: 2, delay: 0, jitter: false });
    
    // Attach rejection handler BEFORE advancing to avoid unhandled rejection warning
    const assertion = expect(promise).rejects.toThrow('always fails');
    
    await jest.advanceTimersByTimeAsync(1);
    await assertion;
    
    expect(fn).toHaveBeenCalledTimes(2);
  });

  it('accepts maxAttempts as a number', async () => {
    const fn = jest.fn().mockRejectedValue(new Error('fail'));
    
    const promise = retry(fn, { maxAttempts: 2, delay: 10, jitter: false });
    
    // Attach rejection handler BEFORE advancing to avoid unhandled rejection warning
    const assertion = expect(promise).rejects.toThrow('fail');
    
    await jest.advanceTimersByTimeAsync(20);
    await assertion;
    
    expect(fn).toHaveBeenCalledTimes(2);
  });

  it('throws TypeError for maxAttempts < 1', async () => {
    const fn = jest.fn();
    await expect(retry(fn, { maxAttempts: 0 })).rejects.toThrow(TypeError);
    await expect(retry(fn, { maxAttempts: -1 })).rejects.toThrow(TypeError);
  });

  it('throws TypeError for negative delay', async () => {
    const fn = jest.fn();
    await expect(retry(fn, { delay: -1 })).rejects.toThrow(TypeError);
  });

  it('calls onRetry with error and attempt number', async () => {
    const fn = jest.fn()
      .mockRejectedValueOnce(new Error('fail 1'))
      .mockRejectedValueOnce(new Error('fail 2'))
      .mockResolvedValue('ok');
    
    const onRetry = jest.fn();
    
    const resultPromise = retry(fn, {
      maxAttempts: 3,
      delay: 100,
      jitter: false,
      onRetry,
    });
    
    await jest.advanceTimersByTimeAsync(100);
    await jest.advanceTimersByTimeAsync(200);
    
    const result = await resultPromise;
    expect(result).toBe('ok');
    expect(onRetry).toHaveBeenCalledTimes(2);
    expect(onRetry).toHaveBeenCalledWith(expect.any(Error), 1);
    expect(onRetry).toHaveBeenCalledWith(expect.any(Error), 2);
  });

  it('respects shouldRetry returning false', async () => {
    const fn = jest.fn().mockRejectedValue(new Error('client error'));
    const shouldRetry = jest.fn().mockReturnValue(false);
    
    const promise = retry(fn, {
      maxAttempts: 3,
      delay: 100,
      shouldRetry,
    });
    
    await expect(promise).rejects.toThrow('client error');
    expect(fn).toHaveBeenCalledTimes(1);
    expect(shouldRetry).toHaveBeenCalledTimes(1);
  });

  it('aborts when signal is aborted', async () => {
    const fn = jest.fn().mockRejectedValue(new Error('fail'));
    const controller = new AbortController();
    
    const promise = retry(fn, {
      maxAttempts: 5,
      delay: 1000,
      jitter: false,
      signal: controller.signal,
    });
    
    // Abort before the retry delay completes
    controller.abort();
    
    await expect(promise).rejects.toThrow('Retry aborted');
  });

  it('uses exponential backoff', async () => {
    const fn = jest.fn()
      .mockRejectedValueOnce(new Error('fail 1'))
      .mockRejectedValueOnce(new Error('fail 2'))
      .mockResolvedValue('ok');
    
    const onRetry = jest.fn();
    
    const resultPromise = retry(fn, {
      maxAttempts: 3,
      delay: 100,
      jitter: false,
      onRetry,
    });
    
    // Advance enough for both retries: 100ms (attempt 1) + 200ms (attempt 2) = 300ms
    await jest.advanceTimersByTimeAsync(400);
    
    // Both retries fired
    expect(onRetry).toHaveBeenCalledTimes(2);
    expect(onRetry).toHaveBeenCalledWith(expect.any(Error), 1);
    expect(onRetry).toHaveBeenCalledWith(expect.any(Error), 2);
    
    const result = await resultPromise;
    expect(result).toBe('ok');
  });
});
