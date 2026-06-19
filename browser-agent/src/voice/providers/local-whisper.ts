/**
 * Local Whisper STT provider.
 *
 * Transcribes audio using a local Whisper installation via subprocess.
 * Supports two backends:
 *   - `whisper` CLI (OpenAI Whisper)
 *   - `faster-whisper` Python package (CTranslate2, faster inference)
 *
 * The caller specifies the backend and binary path in config.
 * This provider does NOT auto-detect or auto-install anything.
 *
 * ⚠️ This provider contains NO business logic.
 * It doesn't know about interviews, resumes, or job descriptions.
 * The caller supplies all behavioral configuration.
 */

import { spawn, type ChildProcess } from 'child_process';
import { readFile, unlink, writeFile } from 'fs/promises';
import { tmpdir } from 'os';
import { join } from 'path';
import { randomBytes } from 'crypto';
import { logger } from '../../utils/logger.js';
import type {
  STTProvider,
  AudioChunk,
  TranscriptSegment,
} from '../types.js';

// ----- Constants -----

const MIN_CHANNELS = 1;
const MAX_CHANNELS = 2;
const MIN_SAMPLE_RATE = 1;
const MAX_SAMPLE_RATE = 192_000;

// ----- Configuration -----

/** Local Whisper STT configuration. */
export interface LocalWhisperConfig {
  /**
   * Path to the whisper binary or Python script.
   * Examples:
   *   - 'whisper' (on PATH)
   *   - '/usr/local/bin/whisper'
   *   - 'python3' (for faster-whisper via module)
   */
  binary: string;
  /**
   * Backend type — determines how arguments are constructed.
   *   - 'whisper': OpenAI Whisper CLI
   *   - 'faster-whisper': CTranslate2-based faster-whisper
   */
  backend: 'whisper' | 'faster-whisper';
  /** Model name (e.g., 'base', 'medium', 'large-v3') */
  model: string;
  /** Language code (e.g., 'en') */
  language?: string;
  /** Additional CLI arguments passed to the binary */
  args?: string[];
  /** Request timeout in ms (default 60000) */
  timeoutMs?: number;
  /** Working directory for the subprocess */
  cwd?: string;
  /** Output directory for temp files (default: os.tmpdir()) */
  outputDir?: string;
}

// ----- Errors -----

/** Error thrown when local Whisper fails. */
export class LocalWhisperError extends Error {
  constructor(
    message: string,
    public readonly exitCode?: number | null,
    cause?: unknown
  ) {
    super(message, { cause });
    this.name = 'LocalWhisperError';
  }
}

// ----- Provider -----

/**
 * Local Whisper speech-to-text provider.
 *
 * Runs Whisper as a subprocess, writes audio to a temp file,
 * reads transcript from stdout or JSON output.
 *
 * Lifecycle: initialize() → transcribe() → destroy()
 */
export class LocalWhisperSTT implements STTProvider {
  readonly name = 'local-whisper';

  private readonly config: LocalWhisperConfig;
  private readonly timeoutMs: number;
  private readonly outputDir: string;
  private readonly log = logger.child({ component: 'LocalWhisperSTT' });
  private initialized = false;
  private destroying = false;
  private activeProcess: ChildProcess | null = null;

  constructor(config: LocalWhisperConfig) {
    this.config = config;
    this.timeoutMs = config.timeoutMs ?? 60_000;
    this.outputDir = config.outputDir ?? tmpdir();
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
      throw new LocalWhisperError('Provider is shutting down');
    }

    try {
      const result = await this.runCommand(['--help']);
      if (result.exitCode !== 0) {
        this.log.error(
          { binary: this.config.binary, exitCode: result.exitCode },
          'Whisper binary not accessible'
        );
        throw new LocalWhisperError(
          `Whisper binary not accessible: ${this.config.binary} (exit ${result.exitCode})`,
          result.exitCode
        );
      }
      this.initialized = true;
    } catch (error) {
      if (error instanceof LocalWhisperError) throw error;
      const err = error instanceof Error ? error : new Error(String(error));
      this.log.error({ binary: this.config.binary, err }, 'Failed to initialize local whisper');
      throw new LocalWhisperError(
        `Failed to initialize local whisper: ${err.message}`,
        undefined,
        error
      );
    }
  }

  /**
   * Transcribe audio chunks to text.
   *
   * Writes audio to a temp WAV file, runs Whisper, reads output.
   * Returns null if transcription is empty.
   *
   * @param audio - Audio chunk to transcribe
   * @returns TranscriptSegment or null if empty
   * @throws LocalWhisperError if subprocess fails or already in progress
   */
  async transcribe(audio: AudioChunk): Promise<TranscriptSegment | null> {
    if (!this.initialized) {
      this.log.error({ method: 'transcribe' }, 'Local Whisper not initialized — call initialize() first');
      throw new LocalWhisperError('Local Whisper not initialized — call initialize() first');
    }

    // Guard: reject concurrent transcription calls
    if (this.activeProcess) {
      this.log.warn({ method: 'transcribe' }, 'Transcription already in progress');
      throw new LocalWhisperError('Transcription already in progress — concurrent calls not supported');
    }

    if (this.destroying) {
      this.log.warn({ method: 'transcribe' }, 'Provider is shutting down');
      throw new LocalWhisperError('Provider is shutting down');
    }

    // Validate input
    if (audio.channels < MIN_CHANNELS || audio.channels > MAX_CHANNELS) {
      this.log.error({ channels: audio.channels }, 'Unsupported channel count');
      throw new LocalWhisperError(`Unsupported channel count: ${audio.channels}`);
    }
    if (audio.sampleRate < MIN_SAMPLE_RATE || audio.sampleRate > MAX_SAMPLE_RATE) {
      this.log.error({ sampleRate: audio.sampleRate }, 'Invalid sample rate');
      throw new LocalWhisperError(`Invalid sample rate: ${audio.sampleRate}`);
    }

    // Write audio to temp file
    const tempId = randomBytes(8).toString('hex');
    const tempWav = join(this.outputDir, `whisper_${tempId}.wav`);
    const tempJson = join(this.outputDir, `whisper_${tempId}.json`);

    try {
      // Convert audio data to WAV buffer
      const wavBuffer = this.pcm16ToWav(
        audio.data instanceof ArrayBuffer
          ? Buffer.from(audio.data)
          : audio.data,
        audio.sampleRate,
        audio.channels
      );

      await writeFile(tempWav, wavBuffer);

      // Build CLI arguments
      const args = this.buildArgs(tempWav, tempJson);
      const result = await this.runCommand(args, this.config.cwd);

      if (result.exitCode !== 0) {
        this.log.error(
          { binary: this.config.binary, exitCode: result.exitCode, stderr: result.stderr },
          'Whisper transcription failed'
        );
        throw new LocalWhisperError(
          `Whisper transcription failed (exit ${result.exitCode}): ${result.stderr}`,
          result.exitCode
        );
      }

      // Parse output — try JSON first, fall back to stdout text
      const text = await this.parseOutput(tempJson, result.stdout);

      if (!text || text.trim().length === 0) {
        this.log.debug({ tempWav }, 'Whisper produced empty transcript');
        return null;
      }

      return {
        speaker: 'user',
        text: text.trim(),
        timestamp: new Date().toISOString(),
      };
    } catch (error) {
      if (error instanceof LocalWhisperError) throw error;
      const err = error instanceof Error ? error : new Error(String(error));
      this.log.error({ tempWav, err }, 'Whisper transcription failed');
      throw new LocalWhisperError(
        `Whisper transcription failed: ${err.message}`,
        undefined,
        error
      );
    } finally {
      // Clean up temp files
      await Promise.all([
        unlink(tempWav).catch(() => {}),
        unlink(tempJson).catch(() => {}),
      ]);
    }
  }

  /**
   * Clean up resources.
   *
   * Kills any active subprocess and waits for exit.
   * Safe to call multiple times — idempotent.
   */
  async destroy(): Promise<void> {
    if (this.destroying) return;
    this.destroying = true;

    if (this.activeProcess) {
      const proc = this.activeProcess;
      this.activeProcess = null;

      // Wait for process to exit (with SIGKILL escalation)
      const exitPromise = new Promise<void>((resolve) => {
        const timer = setTimeout(() => {
          // Escalate to SIGKILL if SIGTERM didn't work
          proc.kill('SIGKILL');
          proc.removeAllListeners();
          resolve();
        }, 5_000);

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
      await exitPromise;
    }

    this.initialized = false;
  }

  // ----- Private helpers -----

  /**
   * Build CLI arguments for the whisper binary.
   *
   * @param inputPath - Path to the input WAV file
   * @param outputPath - Path for JSON output (faster-whisper) or ignored (whisper)
   */
  private buildArgs(inputPath: string, outputPath: string): string[] {
    const args: string[] = [];

    if (this.config.backend === 'faster-whisper') {
      // faster-whisper: python3 -m faster_whisper --model <model> --output-format json <input>
      args.push('-m', 'faster_whisper');
      args.push('--model', this.config.model);
      args.push('--output-format', 'json');
      args.push('--output-dir', this.outputDir);
      if (this.config.language) {
        args.push('--language', this.config.language);
      }
      args.push(inputPath);
    } else {
      // whisper CLI: whisper <input> --model <model> --output_format json
      args.push(inputPath);
      args.push('--model', this.config.model);
      args.push('--output_format', 'json');
      args.push('--output_dir', this.outputDir);
      if (this.config.language) {
        args.push('--language', this.config.language);
      }
    }

    // Append any extra args from config
    if (this.config.args) {
      args.push(...this.config.args);
    }

    return args;
  }

  /**
   * Parse Whisper output — try JSON file first, fall back to stdout.
   */
  private async parseOutput(jsonPath: string, stdout: string): Promise<string> {
    // Try JSON output first (faster-whisper)
    try {
      const jsonContent = await readFile(jsonPath, 'utf8');
      const parsed = JSON.parse(jsonContent) as unknown;

      // Runtime type check — validate JSON shape
      if (
        typeof parsed === 'object' &&
        parsed !== null &&
        'text' in parsed &&
        typeof (parsed as { text: unknown }).text === 'string'
      ) {
        return (parsed as { text: string }).text;
      }

      this.log.debug({ jsonPath }, 'JSON output has unexpected shape, falling back to stdout');
    } catch (err) {
      this.log.debug({ jsonPath, err }, 'JSON output not available, falling back to stdout');
    }

    // Fall back to stdout (whisper CLI prints text to stdout)
    return stdout.trim();
  }

  /**
   * Run a command as a subprocess with timeout.
   *
   * Uses a settled flag to prevent double-resolution from
   * close/error events firing in sequence.
   *
   * @returns Combined stdout, stderr, and exit code
   */
  private runCommand(
    args: string[],
    cwd?: string
  ): Promise<{ stdout: string; stderr: string; exitCode: number | null }> {
    return new Promise((resolve) => {
      const proc = spawn(this.config.binary, args, {
        cwd: cwd ?? this.config.cwd,
        stdio: ['pipe', 'pipe', 'pipe'],
      });

      this.activeProcess = proc;

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
        this.activeProcess = null;
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

  /**
   * Convert PCM16 audio to WAV format.
   *
   * Creates a minimal WAV header for the PCM data.
   * TODO: Extract to voice/utils/wav.ts if more providers need WAV conversion.
   */
  private pcm16ToWav(pcmData: Buffer, sampleRate: number, channels: number): Buffer {
    const bitsPerSample = 16;
    const bytesPerSample = bitsPerSample / 8;
    const blockAlign = channels * bytesPerSample;
    const byteRate = sampleRate * blockAlign;
    const dataSize = pcmData.length;
    const headerSize = 44;

    const header = Buffer.alloc(headerSize);

    // RIFF header
    header.write('RIFF', 0);
    header.writeUInt32LE(36 + dataSize, 4);
    header.write('WAVE', 8);

    // fmt sub-chunk
    header.write('fmt ', 12);
    header.writeUInt32LE(16, 16);
    header.writeUInt16LE(1, 20);
    header.writeUInt16LE(channels, 22);
    header.writeUInt32LE(sampleRate, 24);
    header.writeUInt32LE(byteRate, 28);
    header.writeUInt16LE(blockAlign, 32);
    header.writeUInt16LE(bitsPerSample, 34);

    // data sub-chunk
    header.write('data', 36);
    header.writeUInt32LE(dataSize, 40);

    return Buffer.concat([header, pcmData]);
  }
}

// ----- Factory -----

/**
 * Create a local Whisper STT provider.
 *
 * @param config - Local Whisper configuration
 * @returns Configured STTProvider instance
 */
export function createLocalWhisperSTT(config: LocalWhisperConfig): STTProvider {
  return new LocalWhisperSTT(config);
}
