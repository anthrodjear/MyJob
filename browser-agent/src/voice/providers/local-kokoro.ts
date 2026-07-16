/**
 * Local Kokoro TTS provider.
 *
 * Text-to-speech using a local Kokoro ONNX installation via long-lived daemon.
 * Kokoro is a lightweight, fast neural TTS (82M params) that runs on CPU.
 *
 * Architecture:
 *   - Spawns a single Python daemon process during initialize()
 *   - Daemon loads model ONCE, stays resident for all subsequent calls
 *   - Communication via length-prefixed JSON over stdin/stdout
 *   - No cold starts, no script injection, no orphaned processes
 *
 * The caller supplies the binary path, model paths, and voice in config.
 * This provider does NOT auto-detect or auto-install anything.
 *
 * ⚠️ This provider contains NO business logic.
 * It doesn't know about interviews, resumes, or job descriptions.
 * The caller supplies all behavioral configuration.
 */

import { spawn, type ChildProcess } from 'child_process';
import { readFileSync } from 'fs';
import { join } from 'path';
import { logger } from '../../utils/logger.js';
import type {
  TTSProvider,
  TTSOptions,
  AudioChunk,
} from '../types.js';

// ----- Configuration -----

/** Local Kokoro TTS configuration. */
export interface LocalKokoroConfig {
  /**
   * Path to the Python binary with kokoro-onnx installed.
   * Examples:
   *   - 'python3' (on PATH)
   *   - '/usr/bin/python3.11'
   *   - '/opt/kokoro/venv/bin/python'
   */
  python: string;
  /**
   * Path to the Kokoro ONNX model file.
   * Example: '/models/kokoro-v1.0.onnx'
   */
  modelPath: string;
  /**
   * Path to the voices binary file.
   * Example: '/models/voices-v1.0.bin'
   */
  voicesPath: string;
  /** Voice name (e.g., 'af_nicole', 'af_bella', 'am_adam') */
  voice: string;
  /** Language code (e.g., 'en-us', 'ja-jp') */
  language?: string;
  /** Speech speed (0.5 = slow, 1.0 = normal, 2.0 = fast) */
  speed?: number;
  /** Output sample rate in Hz (default: 24000 — Kokoro's standard output rate) */
  sampleRate?: number;
  /** Request timeout in ms (default 30000) */
  timeoutMs?: number;
}

// ----- Errors -----

/** Error thrown when local Kokoro fails. */
export class LocalKokoroError extends Error {
  constructor(
    message: string,
    public readonly exitCode?: number | null,
    cause?: unknown
  ) {
    super(message, { cause });
    this.name = 'LocalKokoroError';
  }
}

// ----- Constants -----

/** Default Kokoro output sample rate (Hz). */
const DEFAULT_SAMPLE_RATE = 24000;

/** Daemon startup timeout (ms) — model loading is slow on first boot. */
const DAEMON_STARTUP_TIMEOUT_MS = 60_000;

/** Daemon shutdown timeout (ms). */
const DAEMON_SHUTDOWN_TIMEOUT_MS = 5_000;

/** Daemon request timeout (ms). */
const DAEMON_REQUEST_TIMEOUT_MS = 30_000;

/**
 * Output chunk size for streaming reads (bytes).
 *
 * 4096 bytes = 2048 samples at 16-bit PCM.
 * At 24kHz, that's ~85ms of audio per chunk.
 * This balances streaming granularity (not too many yields)
 * against latency (not too large a batch before first yield).
 */
const _CHUNK_SIZE = 4096;

/** Length prefix size in bytes (4-byte little-endian uint32). */
const LENGTH_PREFIX_SIZE = 4;

/**
 * Path to the Python daemon script, relative to this file.
 * __dirname is available in CommonJS modules.
 */
const DAEMON_SCRIPT_PATH = join(__dirname, '..', 'scripts', 'kokoro_stream.py');

// ----- Daemon Protocol -----
//
// Wire format: [4-byte LE length][JSON payload]
//
// Config: passed as CLI arguments to the daemon script
//   --model-path <path>  --voices-path <path>
//
// Request (Node → Python):
//   { "voice": "af_nicole", "speed": 1.0, "lang": "en-us", "text": "Hello world" }
//
// Response (Python → Node), streamed per chunk:
//   { "type": "audio", "data": "<base64 PCM16>", "sample_rate": 24000 }
//   { "type": "audio", "data": "<base64 PCM16>", "sample_rate": 24000 }
//   ...
//   { "type": "done" }

/** Daemon response types. */
interface DaemonAudioResponse {
  type: 'audio';
  data: string;
  sample_rate?: number;
}

interface DaemonDoneResponse {
  type: 'done';
}

interface DaemonErrorResponse {
  type: 'error';
  message: string;
}

interface DaemonReadyResponse {
  type: 'ready';
}

type DaemonResponse = DaemonAudioResponse | DaemonDoneResponse | DaemonErrorResponse | DaemonReadyResponse;

// ----- Provider -----

/**
 * Local Kokoro text-to-speech provider.
 *
 * Spawns a single Python daemon process that loads the model once and stays
 * resident for all synthesis calls. Communication uses length-prefixed JSON
 * over stdin/stdout — no shell injection, no cold starts, no orphaned processes.
 *
 * Lifecycle: initialize() → synthesize() → destroy()
 */
export class LocalKokoroTTS implements TTSProvider {
  readonly name = 'local-kokoro';

  private readonly config: LocalKokoroConfig;
  private readonly sampleRate: number;
  private readonly log = logger.child({ component: 'LocalKokoroTTS' });
  private daemon: ChildProcess | null = null;
  private destroying = false;
  private daemonReady = false;

  constructor(config: LocalKokoroConfig) {
    this.config = config;
    this.sampleRate = config.sampleRate ?? DEFAULT_SAMPLE_RATE;
  }

  /**
   * Initialize the provider.
   *
   * Spawns the Python daemon process with config as CLI arguments.
   * The daemon loads the Kokoro model ONCE — this is the expensive operation.
   * Subsequent synthesize() calls reuse the loaded model with zero cold-start cost.
   */
  async initialize(): Promise<void> {
    if (this.daemonReady) {
      return;
    }

    if (this.destroying) {
      this.log.warn({ method: 'initialize' }, 'Provider is shutting down');
      throw new LocalKokoroError('Provider is shutting down');
    }

    // Validate config before spawning
    if (!this.config.modelPath || !this.config.voicesPath) {
      this.log.error({ modelPath: this.config.modelPath, voicesPath: this.config.voicesPath }, 'Kokoro model_path and voices_path are required');
      throw new LocalKokoroError('model_path and voices_path are required in config');
    }

    // Read the daemon script from the file system
    let scriptContent: string;
    try {
      scriptContent = readFileSync(DAEMON_SCRIPT_PATH, 'utf-8');
    } catch (err) {
      const error = err instanceof Error ? err : new Error(String(err));
      this.log.error({ path: DAEMON_SCRIPT_PATH, err: error }, 'Failed to read daemon script');
      throw new LocalKokoroError(`Failed to read daemon script: ${error.message}`, undefined, error);
    }

    try {
      // Pass config as CLI arguments — no string interpolation into scripts
      const args = [
        '-c', scriptContent,
        '--model-path', this.config.modelPath,
        '--voices-path', this.config.voicesPath,
      ];

      this.daemon = spawn(this.config.python, args, {
        stdio: ['pipe', 'pipe', 'pipe'],
      });

      let stderr = '';
      this.daemon.stderr?.on('data', (chunk: Buffer) => {
        stderr += chunk.toString();
      });

      // Wait for "ready" response with startup timeout
      // (daemon loads model from CLI args, signals when ready)
      const response = await this.readDaemonResponse(DAEMON_STARTUP_TIMEOUT_MS) as DaemonResponse | null;

      if (!response) {
        throw new LocalKokoroError('Daemon failed to start — no response', undefined, stderr);
      }

      if (response.type === 'error') {
        throw new LocalKokoroError(`Daemon init error: ${response.message}`, undefined, stderr);
      }

      if (response.type !== 'ready') {
        throw new LocalKokoroError(`Unexpected daemon response: ${JSON.stringify(response)}`, undefined, stderr);
      }

      this.daemonReady = true;
      this.log.info({ python: this.config.python }, 'Kokoro daemon ready');
    } catch (error) {
      // Clean up the daemon process if init failed
      if (this.daemon) {
        this.daemon.kill('SIGKILL');
        this.daemon = null;
      }
      if (error instanceof LocalKokoroError) throw error;
      const err = error instanceof Error ? error : new Error(String(error));
      this.log.error({ python: this.config.python, err }, 'Failed to initialize kokoro daemon');
      throw new LocalKokoroError(
        `Failed to initialize kokoro daemon: ${err.message}`,
        undefined,
        error
      );
    }
  }

  /**
   * Convert text to audio stream.
   *
   * Sends a synthesis request to the long-lived daemon process.
   * The daemon already has the model loaded — no cold start.
   * Yields AudioChunks as the daemon streams audio back.
   *
   * @param text - Text to synthesize
   * @param options - Voice overrides (voice name, speed)
   * @yields AudioChunk with PCM16 audio data
   * @throws LocalKokoroError if daemon fails or times out
   */
  async *synthesize(text: string, options?: TTSOptions): AsyncIterable<AudioChunk> {
    if (!this.daemonReady || !this.daemon) {
      this.log.error({ method: 'synthesize' }, 'Kokoro daemon not initialized');
      throw new LocalKokoroError('Kokoro daemon not initialized — call initialize() first');
    }

    if (this.destroying) {
      this.log.warn({ method: 'synthesize' }, 'Provider is shutting down');
      throw new LocalKokoroError('Provider is shutting down');
    }

    if (!this.daemon.stdin || !this.daemon.stdout) {
      this.log.error({ method: 'synthesize' }, 'Daemon stdin/stdout not available');
      throw new LocalKokoroError('Daemon stdin/stdout not available');
    }

    const voice = options?.voice ?? this.config.voice;
    const speed = options?.speed ?? this.config.speed ?? 1.0;
    const language = this.config.language ?? 'en-us';

    // Validate speed
    if (typeof speed !== 'number' || !isFinite(speed)) {
      this.log.error({ speed }, 'Invalid speed value');
      throw new LocalKokoroError(`Invalid speed value: ${speed}`);
    }

    // Set up per-request timeout
    let requestTimer: ReturnType<typeof setTimeout> | null = null;
    let timedOut = false;

    try {
      // Send synthesis request
      this.writeDaemonMessage({
        voice,
        speed,
        lang: language,
        text,
      });

      // Set up timeout
      requestTimer = setTimeout(() => {
        timedOut = true;
      }, DAEMON_REQUEST_TIMEOUT_MS);

      // Read streaming responses until "done" or "error"
      while (true) {
        if (timedOut) {
          throw new LocalKokoroError('Synthesis request timed out');
        }

        const response = await this.readDaemonResponse(DAEMON_REQUEST_TIMEOUT_MS) as DaemonResponse | null;

        if (!response) {
          throw new LocalKokoroError('Daemon closed unexpectedly during synthesis');
        }

        if (response.type === 'error') {
          throw new LocalKokoroError(`Daemon synthesis error: ${response.message}`);
        }

        if (response.type === 'done') {
          return;
        }

        if (response.type === 'audio' && response.data) {
          // Decode base64 PCM16 audio — Buffer.from creates a new buffer (no SharedArrayBuffer leak)
          const pcmBuffer = Buffer.from(response.data, 'base64');

          // Yield fixed-size chunks — no Buffer.concat, no O(n²) allocation
          let offset = 0;
          while (offset < pcmBuffer.length) {
            const chunkEnd = Math.min(offset + 4096, pcmBuffer.length);
            yield {
              data: Buffer.from(pcmBuffer.subarray(offset, chunkEnd)),
              sampleRate: response.sample_rate ?? this.sampleRate,
              channels: 1,
            };
            offset = chunkEnd;
          }
        }
      }
    } catch (error) {
      if (timedOut) {
        this.log.error({ text: text.substring(0, 100) }, 'Synthesis request timed out — killing daemon');
        this.killDaemon();
      }
      if (error instanceof LocalKokoroError) throw error;
      const err = error instanceof Error ? error : new Error(String(error));
      this.log.error({ err }, 'Kokoro synthesis failed');
      throw new LocalKokoroError(
        `Kokoro synthesis failed: ${err.message}`,
        undefined,
        error
      );
    } finally {
      if (requestTimer) {
        clearTimeout(requestTimer);
      }
    }
  }

  /**
   * Clean up resources.
   *
   * Sends SIGTERM to the daemon, waits briefly, then escalates to SIGKILL.
   * Safe to call multiple times.
   */
  async destroy(): Promise<void> {
    if (this.destroying) return;
    this.destroying = true;

    this.killDaemon();
    this.daemonReady = false;
  }

  // ----- Private helpers -----

  /**
   * Kill the daemon process with SIGTERM → SIGKILL escalation.
   */
  private killDaemon(): void {
    if (!this.daemon) return;

    const proc = this.daemon;
    this.daemon = null;

    const timer = setTimeout(() => {
      // Escalate to SIGKILL if SIGTERM didn't work
      proc.kill('SIGKILL');
      proc.removeAllListeners();
    }, DAEMON_SHUTDOWN_TIMEOUT_MS);

    proc.on('close', () => {
      clearTimeout(timer);
    });

    proc.on('error', () => {
      clearTimeout(timer);
    });

    proc.kill('SIGTERM');
  }

  /**
   * Write a length-prefixed JSON message to the daemon's stdin.
   *
   * @param obj - Object to serialize and send
   */
  private writeDaemonMessage(obj: Record<string, unknown>): void {
    if (!this.daemon?.stdin) {
      throw new LocalKokoroError('Daemon stdin not available');
    }

    const payload = Buffer.from(JSON.stringify(obj), 'utf-8');
    const lengthPrefix = Buffer.alloc(LENGTH_PREFIX_SIZE);
    lengthPrefix.writeUInt32LE(payload.length, 0);

    this.daemon.stdin.write(lengthPrefix);
    this.daemon.stdin.write(payload);
  }

  /**
   * Read a length-prefixed JSON response from the daemon's stdout.
   *
   * Reads exactly 4 bytes for length, then reads that many bytes for payload.
   * Returns null if the daemon closes.
   *
   * @param timeoutMs - Maximum time to wait for the response
   * @returns Parsed JSON response, or null if daemon closed
   */
  private readDaemonResponse(timeoutMs: number): Promise<DaemonResponse | null> {
    if (!this.daemon?.stdout) {
      return Promise.resolve(null);
    }

    const stdout = this.daemon.stdout;

    return new Promise((resolve) => {
      let settled = false;
      let lengthBuffer: Buffer | null = null;
      let payloadBuffer: Buffer | null = null;
      let payloadBytesRead = 0;

      const timer = setTimeout(() => {
        if (!settled) {
          settled = true;
          stdout.removeListener('data', onData);
          resolve(null);
        }
      }, timeoutMs);

      const cleanup = () => {
        clearTimeout(timer);
        stdout.removeListener('data', onData);
      };

      const onData = (chunk: Buffer) => {
        if (settled) return;

        // Phase 1: Reading length prefix (4 bytes)
        if (lengthBuffer === null) {
          if (chunk.length >= LENGTH_PREFIX_SIZE) {
            lengthBuffer = chunk.subarray(0, LENGTH_PREFIX_SIZE);
            const payloadLength = lengthBuffer.readUInt32LE(0);

            if (payloadLength === 0) {
              // Empty message — skip
              return;
            }

            payloadBuffer = Buffer.alloc(payloadLength);
            payloadBytesRead = 0;

            // Check if the chunk contains the start of the payload
            const available = chunk.length - LENGTH_PREFIX_SIZE;
            if (available > 0) {
              const toCopy = Math.min(available, payloadLength);
              chunk.copy(payloadBuffer, 0, LENGTH_PREFIX_SIZE, LENGTH_PREFIX_SIZE + toCopy);
              payloadBytesRead = toCopy;
            }
          }
          return;
        }

        // Phase 2: Reading payload
        if (payloadBuffer && payloadBytesRead < payloadBuffer.length) {
          const remaining = payloadBuffer.length - payloadBytesRead;
          const toCopy = Math.min(chunk.length, remaining);
          chunk.copy(payloadBuffer, payloadBytesRead, 0, toCopy);
          payloadBytesRead += toCopy;
        }

        // Phase 3: Payload complete — parse and resolve
        if (payloadBuffer && payloadBytesRead >= payloadBuffer.length) {
          if (!settled) {
            settled = true;
            cleanup();
            try {
              const parsed = JSON.parse(payloadBuffer.toString('utf-8')) as DaemonResponse;
              resolve(parsed);
            } catch {
              resolve(null);
            }
          }
        }
      };

      stdout.on('data', onData);
    });
  }
}

// ----- Factory -----

/**
 * Create a local Kokoro TTS provider.
 *
 * @param config - Local Kokoro configuration
 * @returns Configured TTSProvider instance
 */
export function createLocalKokoroTTS(config: LocalKokoroConfig): TTSProvider {
  return new LocalKokoroTTS(config);
}
