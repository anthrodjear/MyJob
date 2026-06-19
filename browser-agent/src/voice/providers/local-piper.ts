/**
 * Local Piper TTS provider.
 *
 * Text-to-speech using a local Piper installation via subprocess.
 * Piper is a fast, local neural TTS that runs on CPU.
 *
 * Architecture:
 *   - Spawns `piper` process per synthesis call
 *   - Pipes text to stdin, reads raw PCM audio from stdout
 *   - Yields AudioChunks as audio arrives (true streaming)
 *
 * The caller supplies the binary path and model in config.
 * This provider does NOT auto-detect or auto-install anything.
 *
 * ⚠️ This provider contains NO business logic.
 * It doesn't know about interviews, resumes, or job descriptions.
 * The caller supplies all behavioral configuration.
 */

import { spawn, type ChildProcess } from 'child_process';
import { logger } from '../../utils/logger.js';
import type {
  TTSProvider,
  TTSOptions,
  AudioChunk,
} from '../types.js';

// ----- Configuration -----

/** Local Piper TTS configuration. */
export interface LocalPiperConfig {
  /**
   * Path to the piper binary.
   * Examples:
   *   - 'piper' (on PATH)
   *   - '/usr/local/bin/piper'
   *   - '/opt/piper/piper'
   */
  binary: string;
  /**
   * Path to the ONNX voice model file.
   * Examples:
   *   - '/models/en_US-lessac-medium.onnx'
   *   - 'en_US-lessac-medium.onnx' (if on PATH)
   */
  model: string;
  /** Sample rate in Hz (default: 22050 — Piper's standard output rate) */
  sampleRate?: number;
  /** Speech length scale (1.0 = normal, <1 = faster, >1 = slower) */
  lengthScale?: number;
  /** Generator noise for variation (0-1, default 0.667) */
  noiseScale?: number;
  /** Phoneme width variation (0-1, default 0.8) */
  noiseW?: number;
  /** Seconds of silence between sentences (default 0.2) */
  sentenceSilence?: number;
  /** Speaker ID for multi-speaker models (default: auto/single) */
  speakerId?: number;
  /** Additional CLI arguments passed to piper */
  args?: string[];
  /** Request timeout in ms (default 30000) */
  timeoutMs?: number;
}

// ----- Errors -----

/** Error thrown when local Piper fails. */
export class LocalPiperError extends Error {
  constructor(
    message: string,
    public readonly exitCode?: number | null,
    cause?: unknown
  ) {
    super(message, { cause });
    this.name = 'LocalPiperError';
  }
}

// ----- Constants -----

/** Default Piper output sample rate (Hz). */
const DEFAULT_SAMPLE_RATE = 22050;

/** Daemon shutdown timeout (ms). */
const SHUTDOWN_TIMEOUT_MS = 5_000;

/** Output chunk size for streaming reads. */
const CHUNK_SIZE = 4096;

// ----- Provider -----

/**
 * Local Piper text-to-speech provider.
 *
 * Streams audio by piping text to piper's stdin and reading PCM from stdout.
 * Each synthesis call spawns a new piper process.
 *
 * Lifecycle: initialize() → synthesize() → destroy()
 */
export class LocalPiperTTS implements TTSProvider {
  readonly name = 'local-piper';

  private readonly config: LocalPiperConfig;
  private readonly sampleRate: number;
  private readonly timeoutMs: number;
  private readonly log = logger.child({ component: 'LocalPiperTTS' });
  private initialized = false;
  private destroying = false;
  private activeProcesses = new Set<ChildProcess>();

  constructor(config: LocalPiperConfig) {
    this.config = config;
    this.sampleRate = config.sampleRate ?? DEFAULT_SAMPLE_RATE;
    this.timeoutMs = config.timeoutMs ?? 30_000;
  }

  /**
   * Initialize the provider.
   *
   * Verifies the binary is accessible by running --help.
   */
  async initialize(): Promise<void> {
    if (this.initialized) {
      return;
    }

    if (this.destroying) {
      this.log.warn({ method: 'initialize' }, 'Provider is shutting down');
      throw new LocalPiperError('Provider is shutting down');
    }

    try {
      const result = await this.runCommand(['--help']);
      if (result.exitCode !== 0) {
        this.log.error(
          { binary: this.config.binary, exitCode: result.exitCode },
          'Piper binary not accessible'
        );
        throw new LocalPiperError(
          `Piper binary not accessible: ${this.config.binary} (exit ${result.exitCode})`,
          result.exitCode
        );
      }
      this.initialized = true;
    } catch (error) {
      if (error instanceof LocalPiperError) throw error;
      const err = error instanceof Error ? error : new Error(String(error));
      this.log.error({ binary: this.config.binary, err }, 'Failed to initialize local piper');
      throw new LocalPiperError(
        `Failed to initialize local piper: ${err.message}`,
        undefined,
        error
      );
    }
  }

  /**
   * Convert text to audio stream.
   *
   * Spawns piper, pipes text to stdin, reads raw PCM from stdout.
   * Yields AudioChunks as audio data arrives — true streaming.
   *
   * @param text - Text to synthesize
   * @param options - Voice overrides (voice maps to speakerId)
   * @yields AudioChunk with PCM16 audio data
   * @throws LocalPiperError if subprocess fails
   */
  async *synthesize(text: string, options?: TTSOptions): AsyncIterable<AudioChunk> {
    if (!this.initialized) {
      this.log.error({ method: 'synthesize' }, 'Local Piper not initialized — call initialize() first');
      throw new LocalPiperError('Local Piper not initialized — call initialize() first');
    }

    if (this.destroying) {
      this.log.warn({ method: 'synthesize' }, 'Provider is shutting down');
      throw new LocalPiperError('Provider is shutting down');
    }

    const args = this.buildArgs(options);

    const proc = spawn(this.config.binary, args, {
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    this.activeProcesses.add(proc);

    let stderr = '';

    proc.stderr?.on('data', (chunk: Buffer) => {
      stderr += chunk.toString();
    });

    // Set up per-request timeout — cleared in finally block
    let timer: ReturnType<typeof setTimeout> | null = null;

    try {
      timer = setTimeout(() => {
        this.log.error({ binary: this.config.binary }, 'Piper synthesis timed out');
        proc.kill('SIGTERM');
      }, this.timeoutMs);

      // Write text to stdin and close — piper reads until EOF
      if (proc.stdin) {
        proc.stdin.write(text);
        proc.stdin.end();
      }

      // Read raw PCM audio from stdout in chunks — no Buffer.concat accumulation
      if (proc.stdout) {
        for await (const chunk of proc.stdout) {
          const chunkBuffer = chunk instanceof Buffer ? chunk : Buffer.from(chunk);

          // Yield fixed-size chunks directly — no O(n²) allocation
          let offset = 0;
          while (offset < chunkBuffer.length) {
            const chunkEnd = Math.min(offset + CHUNK_SIZE, chunkBuffer.length);
            yield {
              data: Buffer.from(chunkBuffer.subarray(offset, chunkEnd)),
              sampleRate: this.sampleRate,
              channels: 1,
            };
            offset = chunkEnd;
          }
        }
      }

      // Wait for process to exit
      const exitCode = await new Promise<number | null>((resolve) => {
        proc.on('close', resolve);
        proc.on('error', () => resolve(null));
      });

      if (exitCode !== 0 && exitCode !== null) {
        this.log.error(
          { binary: this.config.binary, exitCode, stderr },
          'Piper synthesis failed'
        );
        throw new LocalPiperError(
          `Piper synthesis failed (exit ${exitCode}): ${stderr}`,
          exitCode
        );
      }
    } catch (error) {
      if (error instanceof LocalPiperError) throw error;
      const err = error instanceof Error ? error : new Error(String(error));
      this.log.error({ err, stderr }, 'Piper synthesis failed');
      throw new LocalPiperError(
        `Piper synthesis failed: ${err.message}`,
        undefined,
        error
      );
    } finally {
      if (timer) {
        clearTimeout(timer);
      }
      this.activeProcesses.delete(proc);

      // Ensure stdin is closed (in case of early error)
      proc.stdin?.end();
    }
  }

  /**
   * Clean up resources.
   *
   * Kills ALL active piper processes. Safe to call multiple times.
   */
  async destroy(): Promise<void> {
    if (this.destroying) return;
    this.destroying = true;

    const killPromises: Promise<void>[] = [];

    for (const proc of this.activeProcesses) {
      const exitPromise = new Promise<void>((resolve) => {
        const timer = setTimeout(() => {
          // Escalate to SIGKILL if SIGTERM didn't work
          proc.kill('SIGKILL');
          proc.removeAllListeners();
          resolve();
        }, SHUTDOWN_TIMEOUT_MS);

        proc.on('close', () => {
          clearTimeout(timer);
          resolve();
        });

        proc.on('error', () => {
          clearTimeout(timer);
          resolve();
        });
      });

      proc.kill('SIGTERM');
      killPromises.push(exitPromise);
    }

    await Promise.all(killPromises);
    this.activeProcesses.clear();
    this.initialized = false;
  }

  // ----- Private helpers -----

  /**
   * Build CLI arguments for piper.
   *
   * @param options - TTS options (voice → speakerId)
   */
  private buildArgs(options?: TTSOptions): string[] {
    const args: string[] = [];

    // Model (required)
    args.push('--model', this.config.model);

    // Output raw PCM (not WAV) — enables true streaming
    args.push('--output-raw');

    // Length scale (speed)
    if (this.config.lengthScale !== undefined) {
      args.push('--length-scale', String(this.config.lengthScale));
    }

    // Noise parameters
    if (this.config.noiseScale !== undefined) {
      args.push('--noise-scale', String(this.config.noiseScale));
    }
    if (this.config.noiseW !== undefined) {
      args.push('--noise-w', String(this.config.noiseW));
    }

    // Sentence silence
    if (this.config.sentenceSilence !== undefined) {
      args.push('--sentence-silence', String(this.config.sentenceSilence));
    }

    // Speaker ID — options.voice overrides config
    const speakerId = this.resolveSpeakerId(options?.voice);
    if (speakerId !== undefined) {
      args.push('--speaker', String(speakerId));
    }

    // Extra args from config
    if (this.config.args) {
      args.push(...this.config.args);
    }

    return args;
  }

  /**
   * Resolve speaker ID from options or config.
   *
   * @param voiceOverride - Voice string from TTSOptions (parsed as int)
   * @returns Resolved speaker ID, or undefined if not set
   */
  private resolveSpeakerId(voiceOverride?: string): number | undefined {
    // options.voice takes precedence
    if (voiceOverride !== undefined) {
      const parsed = parseInt(voiceOverride, 10);
      if (!isNaN(parsed)) {
        return parsed;
      }
      this.log.warn({ voice: voiceOverride }, 'Could not parse voice as speaker ID');
    }

    // Fall back to config
    return this.config.speakerId;
  }

  /**
   * Run a command as a subprocess (used for init check only).
   *
   * @returns Combined stdout, stderr, and exit code
   */
  private runCommand(
    args: string[]
  ): Promise<{ stdout: string; stderr: string; exitCode: number | null }> {
    return new Promise((resolve) => {
      const proc = spawn(this.config.binary, args, {
        stdio: ['pipe', 'pipe', 'pipe'],
      });

      let stdout = '';
      let stderr = '';
      let settled = false;

      const timer = setTimeout(() => {
        proc.kill('SIGTERM');
      }, this.timeoutMs);

      const settle = (result: { stdout: string; stderr: string; exitCode: number | null }) => {
        if (settled) return;
        settled = true;
        clearTimeout(timer);
        resolve(result);
      };

      proc.stdout?.on('data', (chunk: Buffer) => {
        stdout += chunk.toString();
      });

      proc.stderr?.on('data', (chunk: Buffer) => {
        stderr += chunk.toString();
      });

      proc.on('close', (exitCode) => {
        settle({ stdout, stderr, exitCode });
      });

      proc.on('error', (error) => {
        settle({ stdout, stderr: error.message, exitCode: null });
      });
    });
  }
}

// ----- Factory -----

/**
 * Create a local Piper TTS provider.
 *
 * @param config - Local Piper configuration
 * @returns Configured TTSProvider instance
 */
export function createLocalPiperTTS(config: LocalPiperConfig): TTSProvider {
  return new LocalPiperTTS(config);
}
