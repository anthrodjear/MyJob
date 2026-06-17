import { getLLMConfig, getPrompts, PromptPair } from '../config/config.js';
import { z } from 'zod';
import { logger } from '../utils/logger.js';

// ----- Custom error types -----

/**
 * Error from the Ollama API or transport layer.
 * Preserves the underlying cause for debugging.
 */
export class OllamaError extends Error {
  constructor(message: string, public readonly cause?: unknown) {
    super(message, { cause });
    this.name = 'OllamaError';
  }
}

/**
 * Error extracting or validating structured data from an LLM response.
 */
export class LLMExtractionError extends Error {
  constructor(
    message: string,
    public readonly context: Record<string, unknown> = {},
    cause?: unknown,
  ) {
    super(message, { cause });
    this.name = 'LLMExtractionError';
  }
}

// ----- Zod schemas for LLM wire formats -----

const RemoteTypeSchema = z.enum(['remote', 'hybrid', 'onsite', 'unknown']);

const JobExtractionResultSchema = z.object({
  title: z.string().min(1),
  company: z.string().min(1),
  location: z.string().nullable(),
  remote_type: RemoteTypeSchema,
  salary_min: z.number().int().nonnegative().nullable(),
  salary_max: z.number().int().nonnegative().nullable(),
  salary_currency: z.string().length(3).nullable(),
  requirements: z.string(),
  description: z.string(),
  posted_at: z.string().datetime().nullable(),
  url: z.string().url().nullable(),
});

export type JobExtractionResult = z.infer<typeof JobExtractionResultSchema>;

const OllamaRequestSchema = z.object({
  model: z.string().min(1),
  prompt: z.string().min(1),
  stream: z.literal(false),
  format: z.enum(['json']).optional(),
});

const OllamaResponseSchema = z.object({
  response: z.string(),
  done: z.boolean(),
});

type OllamaRequest = z.infer<typeof OllamaRequestSchema>;
type OllamaResponse = z.infer<typeof OllamaResponseSchema>;

// ----- Constants -----

const DEFAULT_OLLAMA_BASE_URL = 'http://localhost:11434';
const REQUEST_TIMEOUT_MS = 120_000;
const ERROR_BODY_PREVIEW_LEN = 500;

// ----- Client -----

/**
 * Client for communicating with a local Ollama server.
 *
 * Used for LLM-driven tasks: job extraction, form field mapping, etc.
 * Construct directly for tests; use {@link getOllamaClient} for production.
 *
 * @example
 *   const ollama = getOllamaClient();
 *   const result = await ollama.extractJobData(rawHtml);
 */
export class OllamaClient {
  private readonly baseUrl: string;
  private readonly model: string;
  private readonly log = logger.child({ component: 'OllamaClient' });

  constructor(baseUrl?: string, model?: string) {
    if (baseUrl !== undefined && model !== undefined) {
      this.baseUrl = baseUrl;
      this.model = model;
      return;
    }
    // Fall back to config (for direct callers that pass nothing)
    const config = getLLMConfig();
    this.baseUrl = baseUrl ?? config.local.baseUrl ?? DEFAULT_OLLAMA_BASE_URL;
    this.model = model ?? config.local.model;
  }

  /**
   * Call Ollama's /api/generate endpoint with a prompt.
   * Returns the model's text response.
   * @throws OllamaError on transport or API failures.
   */
  async generate(prompt: string): Promise<string> {
    const requestBody: OllamaRequest = {
      model: this.model,
      prompt,
      stream: false,
    };

    this.log.debug({ model: this.model, promptLen: prompt.length }, 'Ollama request');

    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);

    let response: Response;
    try {
      response = await fetch(`${this.baseUrl}/api/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(requestBody),
        signal: controller.signal,
      });
    } catch (e) {
      this.log.error({ err: e, model: this.model }, 'Ollama transport error');
      throw new OllamaError(`Failed to call Ollama at ${this.baseUrl}`, e);
    } finally {
      clearTimeout(timer);
    }

    if (!response.ok) {
      const errorText = await response.text();
      const preview = errorText.slice(0, ERROR_BODY_PREVIEW_LEN);
      this.log.error({ status: response.status, body: preview }, 'Ollama API error');
      throw new OllamaError(`Ollama API error ${response.status}: ${preview}`);
    }

    const rawJson = await response.json();
    const parsed = OllamaResponseSchema.safeParse(rawJson);
    if (!parsed.success) {
      this.log.error({ issues: parsed.error.issues }, 'Invalid Ollama response shape');
      throw new OllamaError('Invalid Ollama response shape');
    }

    return parsed.data.response;
  }

  /**
   * Call Ollama with the `job_extraction` prompt and parse the JSON response.
   * @param rawContent - The raw job posting text/HTML to extract from.
   * @throws LLMExtractionError if the LLM output cannot be parsed or validated.
   */
  async extractJobData(rawContent: string): Promise<JobExtractionResult> {
    if (!rawContent || rawContent.trim().length === 0) {
      throw new LLMExtractionError('Cannot extract from empty content');
    }

    const prompts = getPrompts();
    const prompt = this.buildPrompt(prompts.job_extraction, { RawContent: rawContent });

    return this.generateStructured(prompt, JobExtractionResultSchema);
  }

  /**
   * Extract the first balanced JSON object from a string that may contain
   * explanatory text, markdown fences, or other commentary.
   * Useful for parsing LLM responses that include extra commentary.
   */
  public extractFirstJsonObject(input: string): string {
    // Strip markdown code fences
    const stripped = input
      .replace(/^```(?:json)?\s*\n?/i, '')
      .replace(/\n?```\s*$/i, '');

    const start = stripped.indexOf('{');
    if (start === -1) {
      throw new LLMExtractionError('No JSON object found in LLM response', {
        raw: stripped.slice(0, ERROR_BODY_PREVIEW_LEN),
      });
    }

    let depth = 0;
    let inString = false;
    let escape = false;
    for (let i = start; i < stripped.length; i++) {
      const c = stripped[i];
      if (escape) {
        escape = false;
        continue;
      }
      if (inString && c === '\\') {
        escape = true;
        continue;
      }
      if (c === '"') {
        inString = !inString;
        continue;
      }
      if (inString) continue;
      if (c === '{') depth++;
      else if (c === '}') {
        depth--;
        if (depth === 0) return stripped.slice(start, i + 1);
      }
    }
    throw new LLMExtractionError('Unbalanced JSON in LLM response', {
      raw: stripped.slice(start, start + ERROR_BODY_PREVIEW_LEN),
    });
  }

  /**
   * Replace Go-style template placeholders ({{.Key}}) in both system and user prompts.
   * Public so other modules can reuse the same template substitution logic.
   */
  public buildPrompt(promptPair: PromptPair, data: Record<string, string | number | boolean | null>): string {
    let systemPrompt = promptPair.system;
    let userPrompt = promptPair.user;

    for (const [key, value] of Object.entries(data)) {
      const placeholder = `{{.${key}}}`;
      const stringValue =
        typeof value === 'string' ? value : String(value);
      systemPrompt = systemPrompt.replaceAll(placeholder, stringValue);
      userPrompt = userPrompt.replaceAll(placeholder, stringValue);
    }

    return `${systemPrompt}\n\n${userPrompt}`;
  }

  /**
   * Generate a structured response from the LLM, validated against a Zod schema.
   *
   * Combines `generate()` → `extractFirstJsonObject()` → `JSON.parse()` → schema validation
   * into a single call. Every LLM consumer should use this instead of repeating the pattern.
   *
   * @param prompt - The full prompt to send (use `buildPrompt()` to construct it).
   * @param schema - Zod schema to validate the parsed JSON against.
   * @returns Validated data matching the schema.
   * @throws LLMExtractionError if extraction or validation fails.
   */
  async generateStructured<T>(prompt: string, schema: z.ZodSchema<T>): Promise<T> {
    const response = await this.generate(prompt);

    let jsonStr: string;
    try {
      jsonStr = this.extractFirstJsonObject(response);
    } catch (e) {
      throw new LLMExtractionError(
        'No JSON object found in LLM response',
        { promptLen: prompt.length, responseLen: response.length },
        e,
      );
    }

    let parsedJson: unknown;
    try {
      parsedJson = JSON.parse(jsonStr);
    } catch (e) {
      throw new LLMExtractionError(
        'LLM response was not valid JSON',
        { raw: jsonStr.slice(0, ERROR_BODY_PREVIEW_LEN) },
        e,
      );
    }

    const validated = schema.safeParse(parsedJson);
    if (!validated.success) {
      throw new LLMExtractionError(
        'LLM response failed schema validation',
        {
          issues: validated.error.issues,
          raw: jsonStr.slice(0, ERROR_BODY_PREVIEW_LEN),
        },
      );
    }

    return validated.data;
  }
}

// ----- Singleton -----

let ollamaClientInstance: OllamaClient | null = null;

/**
 * Returns the process-wide OllamaClient singleton.
 * Use direct construction in tests.
 */
export function getOllamaClient(): OllamaClient {
  if (ollamaClientInstance === null) {
    ollamaClientInstance = new OllamaClient();
  }
  return ollamaClientInstance;
}

/**
 * Clear the singleton. Useful for tests.
 */
export function clearOllamaClient(): void {
  ollamaClientInstance = null;
}