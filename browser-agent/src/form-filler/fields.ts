import { getOllamaClient, LLMExtractionError } from '../llm/ollama.js';
import { getPrompts } from '../config/config.js';
import { FormField } from './detector.js';
import { logger } from '../utils/logger.js';
import { z } from 'zod';

/**
 * A mapping from a form field to a candidate data value.
 *
 * @property confidence - LLM-assigned confidence (0–1). Heuristic fallback uses fixed tiers.
 */
export interface FieldMapping {
  /** ID or selector of the target form field. */
  field_id: string;
  /** Human-readable label of the target field. */
  field_label: string;
  /** Key in the candidate data object. */
  candidate_data_key: string;
  /** The value to fill into the field. */
  value: string;
  /** Confidence score (0–1). */
  confidence: number;
}

/** Zod schema for LLM field mapping response validation. */
const FieldMappingResultSchema = z.object({
  mappings: z.array(z.object({
    field_id: z.string().min(1),
    field_label: z.string().min(1),
    candidate_data_key: z.string().min(1),
    value: z.string(),
    confidence: z.number().min(0).max(1),
  })),
});

/**
 * Error thrown when LLM-based field mapping fails.
 * Extends `LLMExtractionError` for consistent error handling.
 */
export class FieldMappingError extends LLMExtractionError {
  constructor(message: string, context: Record<string, unknown> = {}, cause?: unknown) {
    super(message, context, cause);
    this.name = 'FieldMappingError';
  }
}

/**
 * Normalize a string for fuzzy matching: split camelCase/PascalCase, lowercase, collapse whitespace.
 * `"phoneNumber"` → `"phone number"`, `"firstName"` → `"first name"`.
 */
function normalizeForMatch(str: string): string {
  return str
    .replace(/([a-z])([A-Z])/g, '$1 $2')
    .toLowerCase()
    .replace(/\s+/g, ' ')
    .trim();
}

/**
 * Map form fields to candidate data using LLM-based understanding.
 *
 * Calls Ollama with the `form_understanding` prompt, validates the response
 * against a Zod schema, and falls back to heuristic matching on failure.
 *
 * @param fields - Detected form fields from `detectFormFields`.
 * @param candidateData - Candidate profile data (name, email, experience, etc.).
 * @returns Array of field mappings with confidence scores.
 */
export async function mapFieldsToCandidateData(
  fields: FormField[],
  candidateData: Record<string, unknown>,
): Promise<FieldMapping[]> {
  const ollama = getOllamaClient();
  const prompts = getPrompts();

  // Prepare form fields for LLM
  const formFieldsJson = JSON.stringify(fields.map(f => ({
    id: f.id,
    name: f.name,
    type: f.type,
    label: f.label,
    placeholder: f.placeholder,
    required: f.required,
    options: f.options,
  })));

  // Prepare candidate data for LLM
  const candidateDataJson = JSON.stringify(candidateData);

  // Build prompt using shared template substitution
  const prompt = ollama.buildPrompt(prompts.form_understanding, {
    FormFields: formFieldsJson,
    CandidateData: candidateDataJson,
  });

  try {
    // Single call: generate → extract JSON → parse → validate schema
    const result = await ollama.generateStructured(prompt, FieldMappingResultSchema);
    return result.mappings;
  } catch (error) {
    // Only catch LLM/extraction failures — let programming errors propagate
    if (error instanceof LLMExtractionError) {
      logger.warn(
        {
          err: error,
          fieldCount: fields.length,
          candidateKeyCount: Object.keys(candidateData).length,
        },
        'LLM field mapping failed, falling back to heuristic',
      );
      return heuristicMap(fields, candidateData);
    }
    throw error;
  }
}

// ── Heuristic engine (static patterns, resolved keys injected per call) ──────

/**
 * Priority rules for common field types.
 * `keyTargets` are resolved to actual candidate keys at call time.
 * First match wins (highest priority).
 */
const FIELD_PATTERNS: Array<{
  match: (searchText: string) => boolean;
  keyTargets: string[];
  confidence: number;
}> = [
  { match: (s) => s.includes('email'), keyTargets: ['email', 'emailaddress'], confidence: 0.95 },
  { match: (s) => s.includes('phone'), keyTargets: ['phone', 'phonenumber'], confidence: 0.95 },
  { match: (s) => s.includes('linkedin'), keyTargets: ['linkedin'], confidence: 0.95 },
  { match: (s) => s.includes('github'), keyTargets: ['github'], confidence: 0.95 },
  { match: (s) => s.includes('resume') || s.includes('cv') || s.includes('file'),
    keyTargets: ['resume', 'cv'], confidence: 0.9 },
  { match: (s) => s.includes('name') && !s.includes('username') && !s.includes('company'),
    keyTargets: ['name', 'fullname'], confidence: 0.85 },
  { match: (s) => s.includes('address'), keyTargets: ['address'], confidence: 0.8 },
  { match: (s) => s.includes('city'), keyTargets: ['city'], confidence: 0.8 },
  { match: (s) => s.includes('state') || s.includes('province'),
    keyTargets: ['state', 'province'], confidence: 0.8 },
  { match: (s) => s.includes('zip') || s.includes('postal'),
    keyTargets: ['zip', 'postal', 'zipcode'], confidence: 0.8 },
  { match: (s) => s.includes('country'), keyTargets: ['country'], confidence: 0.8 },
  { match: (s) => s.includes('company'), keyTargets: ['company', 'employer'], confidence: 0.8 },
  { match: (s) => s.includes('title') || s.includes('position'),
    keyTargets: ['title', 'position', 'role'], confidence: 0.8 },
  { match: (s) => s.includes('year') || s.includes('experience'),
    keyTargets: ['experience', 'years'], confidence: 0.75 },
  { match: (s) => s.includes('skill'), keyTargets: ['skills', 'technologies'], confidence: 0.75 },
  { match: (s) => s.includes('education') || s.includes('degree') || s.includes('school'),
    keyTargets: ['education', 'degree', 'university'], confidence: 0.75 },
  { match: (s) => s.includes('summary') || s.includes('objective') || s.includes('about'),
    keyTargets: ['summary', 'objective', 'about'], confidence: 0.7 },
];

/**
 * Heuristic field-to-candidate data mapping.
 *
 * Uses priority-based pattern matching on field label/name/id/placeholder.
 * Returns empty string for unmatched fields (not null) to keep the
 * mapping array uniform.
 *
 * @internal Used as fallback when LLM extraction fails.
 */
function heuristicMap(
  fields: FormField[],
  candidateData: Record<string, unknown>,
): FieldMapping[] {
  const mappings: FieldMapping[] = [];
  const keys = Object.keys(candidateData);

  // Build resolved patterns once per call (keyTargets → actual keys)
  const resolvedPatterns = FIELD_PATTERNS.map(p => ({
    match: p.match,
    key: resolveKey(keys, p.keyTargets),
    confidence: p.confidence,
  }));

  // Track which candidate keys have been used (for deduplication)
  const usedKeys = new Set<string>();

  for (const field of fields) {
    const searchText = normalizeForMatch(
      `${field.label} ${field.name} ${field.id} ${field.placeholder}`
    );

    // Try priority patterns first
    let bestMatch: { key: string; confidence: number } | null = null;

    for (const pattern of resolvedPatterns) {
      if (pattern.key && pattern.match(searchText)) {
        bestMatch = { key: pattern.key, confidence: pattern.confidence };
        break; // First match wins (highest priority)
      }
    }

    // Fallback: fuzzy match on any candidate key (normalized for camelCase)
    if (!bestMatch) {
      for (const key of keys) {
        const normalizedKey = normalizeForMatch(key);
        if (searchText.includes(normalizedKey) || normalizedKey.includes(searchText)) {
          bestMatch = { key, confidence: 0.6 };
          break;
        }
      }
    }

    if (bestMatch) {
      // Skip if this candidate key was already mapped (dedup)
      if (usedKeys.has(bestMatch.key)) continue;
      usedKeys.add(bestMatch.key);

      mappings.push({
        field_id: field.id || field.selector,
        field_label: field.label,
        candidate_data_key: bestMatch.key,
        value: String(candidateData[bestMatch.key] ?? ''),
        confidence: bestMatch.confidence,
      });
    }
  }

  return mappings;
}

/**
 * Find the first candidate key that matches any of the target names.
 * Case-insensitive. Returns the original key casing if found, empty string otherwise.
 */
function resolveKey(keys: string[], targets: string[]): string {
  const lowerKeys = keys.map(k => k.toLowerCase());
  for (const target of targets) {
    const idx = lowerKeys.indexOf(target);
    if (idx !== -1) return keys[idx];
  }
  return '';
}
