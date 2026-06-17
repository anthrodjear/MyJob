/**
 * Structured logger for the browser agent.
 *
 * Uses console methods but produces JSON output for log aggregation.
 * Future: can be swapped for pino/winston without changing call sites.
 *
 * @example
 *   const log = logger.child({ component: 'OllamaClient' });
 *   log.info({ model: 'qwen2.5' }, 'Starting generation');
 */

/** Supported log levels in order of severity. */
export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

/** Structured log fields — any serializable key-value pairs. */
export type LogFields = Record<string, unknown>;

/** Structured logger interface matching zap.Logger capabilities. */
export interface Logger {
  debug(fields: LogFields, msg?: string): void;
  info(fields: LogFields, msg?: string): void;
  warn(fields: LogFields, msg?: string): void;
  error(fields: LogFields, msg?: string): void;
  /**
   * Create a child logger with additional bindings.
   *
   * Bindings are shallow-merged: child keys overwrite parent keys at the
   * top level only. Nested objects are replaced, not deep-merged.
   */
  child(bindings: LogFields): Logger;
}

const LEVELS: Record<LogLevel, number> = {
  debug: 10,
  info: 20,
  warn: 30,
  error: 40,
};

/** Lazily resolved active log level. Avoids module-load-time side effects. */
let _activeLevel: number | undefined;

/**
 * Parse and validate LOG_LEVEL environment variable.
 * Returns 'info' if unset or invalid.
 */
function parseLogLevel(): LogLevel {
  const raw = process.env.LOG_LEVEL?.toLowerCase();
  if (raw && raw in LEVELS) {
    return raw as LogLevel;
  }
  if (raw) {
    // eslint-disable-next-line no-console
    console.error(`[logger] Invalid LOG_LEVEL="${raw}", defaulting to "info"`);
  }
  return 'info';
}

/** Get the active log level (lazily initialized). */
function getActiveLevel(): number {
  if (_activeLevel === undefined) {
    _activeLevel = LEVELS[parseLogLevel()];
  }
  return _activeLevel;
}

/**
 * Reset the cached log level. For testing only.
 * @internal
 */
export function _resetLevelForTesting(): void {
  _activeLevel = undefined;
}

/** Check if a level should be logged. */
function shouldLog(level: LogLevel): boolean {
  return LEVELS[level] >= getActiveLevel();
}

/**
 * Serialize Error objects preserving stack traces and causes.
 * @internal
 */
function serializeError(err: unknown): Record<string, unknown> {
  if (err instanceof Error) {
    return {
      name: err.name,
      message: err.message,
      stack: err.stack,
      cause: err.cause ? serializeError(err.cause) : undefined,
    };
  }
  return { value: String(err) };
}

/**
 * Deep-serialize objects, handling Errors, arrays, and circular references.
 * Primitives pass through unchanged. Objects are recursively serialized.
 * @internal
 */
function deepSerialize(obj: unknown, seen?: WeakSet<object>): unknown {
  if (obj instanceof Error) return serializeError(obj);
  if (obj && typeof obj === 'object') {
    const currentSeen = seen ?? new WeakSet<object>();
    if (currentSeen.has(obj)) return '[Circular]';
    currentSeen.add(obj);

    if (Array.isArray(obj)) {
      return obj.map((item) => deepSerialize(item, currentSeen));
    }

    const out: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(obj)) {
      out[k] = deepSerialize(v, currentSeen);
    }
    return out;
  }
  return obj;
}

/**
 * Emit a log entry as a single JSON line.
 * warn+error go to stderr; info+debug go to stdout.
 * @internal
 */
function emit(
  level: LogLevel,
  bindings: LogFields,
  fields: LogFields,
  msg: string | undefined,
): void {
  if (!shouldLog(level)) return;

  const serializedBindings = deepSerialize(bindings) as Record<string, unknown>;
  const serializedFields = deepSerialize(fields) as Record<string, unknown>;

  const entry = {
    level,
    time: new Date().toISOString(),
    ...serializedBindings,
    ...serializedFields,
    msg: msg ?? '',
  };

  const stream = level === 'error' ? console.error
    : level === 'warn' ? console.warn
    : console.log;
  stream(JSON.stringify(entry));
}

class ConsoleLogger implements Logger {
  constructor(private readonly bindings: LogFields = {}) {}

  debug(fields: LogFields, msg?: string): void {
    emit('debug', this.bindings, fields, msg);
  }

  info(fields: LogFields, msg?: string): void {
    emit('info', this.bindings, fields, msg);
  }

  warn(fields: LogFields, msg?: string): void {
    emit('warn', this.bindings, fields, msg);
  }

  error(fields: LogFields, msg?: string): void {
    emit('error', this.bindings, fields, msg);
  }

  child(bindings: LogFields): Logger {
    return new ConsoleLogger({ ...this.bindings, ...bindings });
  }
}

export const logger: Logger = new ConsoleLogger({ service: 'browser-agent' });
