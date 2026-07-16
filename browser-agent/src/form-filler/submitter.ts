import { Page } from 'playwright';
import { access } from 'node:fs/promises';
import { FormField } from './detector.js';
import { FieldMapping } from './fields.js';
import { logger } from '../utils/logger.js';

const log = logger.child({ component: 'FormSubmitter' });

// ── Constants ─────────────────────────────────────────────────────

const FIELD_WAIT_TIMEOUT_MS = 5_000;
const SUBMIT_VISIBILITY_TIMEOUT_MS = 5_000;
const SUBMIT_NAV_TIMEOUT_MS = 15_000;

// ── CSS selector escaping (Node.js has no CSS.escape) ─────────────

/**
 * Escape a string for use as a CSS ID selector.
 * Handles leading digits (invalid CSS identifiers) and special characters.
 * Implementation mirrors CSS.escape() behavior.
 */
function escapeId(value: string): string {
  if (value === '') return '';
  let result = '';
  const codePoints = Array.from(value);
  for (let i = 0; i < codePoints.length; i++) {
    const ch = codePoints[i];
    const code = ch.codePointAt(0)!;
    if (i === 0) {
      if (code >= 0x30 && code <= 0x39) {
        result += '\\' + code.toString(16).toUpperCase() + ' ';
        continue;
      }
      if (code === 0x2D) {
        const nextCode = codePoints[1]?.codePointAt(0);
        if (nextCode === 0x2D || (nextCode !== undefined && nextCode >= 0x30 && nextCode <= 0x39)) {
          result += '\\' + code.toString(16).toUpperCase() + ' ';
          continue;
        }
      }
    }
    if (
      code <= 0x1F || code === 0x7F ||
      code === 0x20 || code === 0x22 || code === 0x23 ||
      code === 0x24 || code === 0x25 || code === 0x26 ||
      code === 0x27 || code === 0x28 || code === 0x29 ||
      code === 0x2A || code === 0x2B || code === 0x2C ||
      code === 0x2D || code === 0x2E || code === 0x2F ||
      code === 0x3A || code === 0x3B || code === 0x3C ||
      code === 0x3D || code === 0x3E || code === 0x3F ||
      code === 0x40 || code === 0x5B || code === 0x5C ||
      code === 0x5D || code === 0x5E || code === 0x60 ||
      code === 0x7B || code === 0x7C || code === 0x7D
    ) {
      result += '\\' + code.toString(16).toUpperCase() + ' ';
    } else {
      result += ch;
    }
  }
  return result;
}

/**
 * Escape a string for use inside a quoted CSS attribute value.
 * Mirrors detector.ts escapeAttr for consistency.
 */
function escapeAttr(value: string): string {
  return value
    .replace(/\\/g, '\\\\')
    .replace(/"/g, '\\"')
    .replace(/\[/g, '\\[')
    .replace(/\]/g, '\\]')
    .replace(/,/g, '\\,')
    .replace(/:/g, '\\:');
}

/**
 * Build a CSS selector for a form field, preferring selector -> #id -> [name=].
 * Uses the same escaping logic as the detector for consistency.
 */
function buildSelector(field: FormField): string {
  if (field.selector) return field.selector;
  if (field.id) return `#${escapeId(field.id)}`;
  if (field.name) return `[name="${escapeAttr(field.name)}"]`;
  return '';
}

// ── File upload types ─────────────────────────────────────────────

/**
 * Options for file upload, supporting resume, cover letter, and portfolio.
 * Each path is optional — only matched file inputs will receive uploads.
 */
export interface FileUploadOptions {
  /** Path to the resume PDF file. */
  resumePath?: string;
  /** Path to the cover letter PDF file. */
  coverLetterPath?: string;
  /** Path to the portfolio file (PDF, ZIP, etc.). */
  portfolioPath?: string;
}

// ── Types ──────────────────────────────────────────────────────────

/**
 * Result of filling a form on a page.
 */
export interface FillResult {
  /** Whether all fields were filled without errors. */
  success: boolean;
  /** Number of fields successfully filled. */
  filledFields: number;
  /** Any errors encountered during filling. */
  errors: string[];
  /** Screenshot of the filled form (PNG), if capture succeeded. */
  screenshot?: Buffer;
}

// ── Main orchestrator ──────────────────────────────────────────────

/**
 * Fill all mapped form fields on the page and optionally upload files.
 *
 * @param page - Playwright page with the form loaded.
 * @param fields - Detected form fields (from detectFormFields).
 * @param mappings - Field mappings (from mapFieldsToCandidateData).
 * @param fileUploads - Optional file upload paths (resume, cover letter, portfolio).
 * @returns Result with filled count, errors, and optional screenshot.
 */
export async function fillForm(
  page: Page,
  fields: FormField[],
  mappings: FieldMapping[],
  fileUploads?: FileUploadOptions,
): Promise<FillResult> {
  const errors: string[] = [];
  let filledCount = 0;

  // Create a map for quick lookup - use both id and selector for grouped fields
  const mappingMap = new Map(mappings.map(m => [m.field_id, m]));

  for (const field of fields) {
    // For grouped fields (radio/checkbox), field.id is empty, use selector
    const lookupKey = field.id || field.selector;
    const mapping = mappingMap.get(lookupKey);
    if (!mapping || !mapping.value) continue;

    try {
      await fillField(page, field, mapping.value);
      filledCount++;
    } catch (error) {
      const errMsg = error instanceof Error ? error.message : String(error);
      const msg = `Field ${field.label} (${field.id || field.selector}): ${errMsg}`;
      errors.push(msg);
      log.warn(
        { fieldId: field.id, fieldType: field.type, selector: field.selector, err: errMsg },
        'Failed to fill field',
      );
    }
  }

  // Handle file uploads
  if (fileUploads) {
    try {
      await uploadFiles(page, fields, fileUploads);
    } catch (error) {
      const errMsg = error instanceof Error ? error.message : String(error);
      errors.push(`File upload: ${errMsg}`);
    }
  }

  // Take screenshot for verification — non-critical, log on failure
  let screenshot: Buffer | undefined;
  try {
    screenshot = await page.screenshot({ fullPage: true, type: 'png' });
  } catch (err) {
    const errMsg = err instanceof Error ? err.message : String(err);
    log.warn({ err: errMsg }, 'Failed to capture form screenshot');
  }

  return {
    success: errors.length === 0,
    filledFields: filledCount,
    errors,
    screenshot,
  };
}

// ── Single field fill ──────────────────────────────────────────────

/**
 * Fill a single form field with the given value.
 *
 * Supports all standard input types, textarea, select, checkbox, and radio.
 * Throws on unsupported field types or if the field is not visible.
 */
async function fillField(page: Page, field: FormField, value: string): Promise<void> {
  const selector = buildSelector(field);
  if (!selector) {
    throw new Error(`No selector available for field "${field.label}"`);
  }

  const element = page.locator(selector).first();

  // Wait for element to be visible — throw descriptive error on timeout
  try {
    await element.waitFor({ state: 'visible', timeout: FIELD_WAIT_TIMEOUT_MS });
  } catch {
    throw new Error(
      `Field "${field.label}" not visible within ${FIELD_WAIT_TIMEOUT_MS}ms (selector: ${selector})`,
    );
  }

  switch (field.type) {
    case 'text':
    case 'email':
    case 'tel':
    case 'url':
    case 'number':
    case 'textarea':
    case 'search':
    case 'password':
    case 'date':
    case 'datetime-local':
    case 'time':
    case 'month':
    case 'week':
    case 'color':
      await element.fill(value);
      break;

    case 'select':
      try {
        await element.selectOption({ value });
      } catch (firstErr) {
        try {
          await element.selectOption({ label: value });
        } catch {
          const errMsg = firstErr instanceof Error ? firstErr.message : String(firstErr);
          throw new Error(
            `Option "${value}" not found by value or label in field "${field.label}": ${errMsg}`,
          );
        }
      }
      break;

    case 'checkbox': {
      const checkValue = value.toLowerCase();
      if (checkValue === 'true' || checkValue === '1' || checkValue === 'yes' || checkValue === 'on') {
        await element.check();
      } else {
        await element.uncheck();
      }
      break;
    }

    case 'radio':
      await element.check();
      break;

    case 'file':
      // Handled separately in uploadFiles
      break;

    default:
      throw new Error(`Unsupported field type: ${field.type}`);
  }
}

// ── File upload (smart field matching) ─────────────────────────────

/**
 * Regex patterns for identifying file input field types.
 * Each pattern matches against the field's label, name, id, and placeholder.
 */
const FILE_FIELD_PATTERNS = {
  resume: /resume|cv|curriculum\s*vitae/i,
  coverLetter: /cover\s*letter|coverletter/i,
  portfolio: /portfolio|work\s*sample|attachment|document|file|upload/i,
} as const;

/**
 * Identify what type of file a file input field expects.
 * Returns 'resume', 'coverLetter', 'portfolio', or 'unknown'.
 */
function identifyFileType(field: FormField): string {
  const searchText = `${field.label} ${field.name} ${field.id} ${field.placeholder}`.toLowerCase();

  if (FILE_FIELD_PATTERNS.resume.test(searchText)) return 'resume';
  if (FILE_FIELD_PATTERNS.coverLetter.test(searchText)) return 'coverLetter';
  // Portfolio is the broadest pattern — only match if no resume/cover letter
  if (FILE_FIELD_PATTERNS.portfolio.test(searchText)) return 'portfolio';

  return 'unknown';
}

/**
 * Upload files to file input fields on the page.
 * Matches file types to fields based on label/name/id patterns.
 * Falls back to uploading resume if field type cannot be determined.
 *
 * @param page - Playwright page with the form loaded.
 * @param fields - Detected form fields (from detectFormFields).
 * @param fileUploads - File upload paths for resume, cover letter, portfolio.
 */
async function uploadFiles(page: Page, fields: FormField[], fileUploads: FileUploadOptions): Promise<void> {
  // Verify files exist before attempting upload
  const filesToVerify: Array<{ path: string; type: string }> = [];
  if (fileUploads.resumePath) filesToVerify.push({ path: fileUploads.resumePath, type: 'resume' });
  if (fileUploads.coverLetterPath) filesToVerify.push({ path: fileUploads.coverLetterPath, type: 'cover letter' });
  if (fileUploads.portfolioPath) filesToVerify.push({ path: fileUploads.portfolioPath, type: 'portfolio' });

  for (const file of filesToVerify) {
    try {
      await access(file.path);
    } catch {
      throw new Error(`${file.type} file not found: ${file.path}`);
    }
  }

  const fileInputs = fields.filter(f => f.type === 'file');
  if (fileInputs.length === 0) {
    log.warn({ fileInputCount: 0, resumePath: fileUploads.resumePath }, 'No file inputs found on page');
    return;
  }

  // Track which files have been uploaded to avoid duplicates
  const uploadedTypes = new Set<string>();

  for (const field of fileInputs) {
    const selector = buildSelector(field);
    if (!selector) {
      log.warn({ fieldId: field.id, fieldLabel: field.label }, 'File input has no selector, skipping');
      continue;
    }

    const fieldType = identifyFileType(field);
    let filePath: string | undefined;

    // Map identified type to available file
    switch (fieldType) {
      case 'resume':
        filePath = fileUploads.resumePath;
        break;
      case 'coverLetter':
        filePath = fileUploads.coverLetterPath;
        break;
      case 'portfolio':
        filePath = fileUploads.portfolioPath;
        break;
      case 'unknown':
        // Fallback: use resume if not yet uploaded, otherwise skip
        // Track resume as used regardless of field type to prevent duplicates
        if (!uploadedTypes.has('resume') && fileUploads.resumePath) {
          filePath = fileUploads.resumePath;
          uploadedTypes.add('resume'); // Prevent other unknown fields from also getting resume
        }
        break;
    }

    if (filePath) {
      await page.locator(selector).first().setInputFiles(filePath);
      uploadedTypes.add(fieldType);
      log.debug({ fieldLabel: field.label, fieldType, filePath }, 'Uploaded file to field');
    }
  }
}

// ── Submit ─────────────────────────────────────────────────────────

/**
 * Find and click the submit button on the form.
 *
 * Tries common submit selectors: button[type=submit], input[type=submit],
 * and buttons with text like "Submit", "Apply", "Send", "Continue", "Next".
 *
 * After clicking, waits for navigation or URL change instead of networkidle
 * to avoid hangs on pages with persistent connections (analytics, WebSockets).
 *
 * @returns true if a submit button was found and clicked, false otherwise.
 */
export async function submitForm(page: Page): Promise<boolean> {
  const submitSelectors = [
    'button[type="submit"]',
    'input[type="submit"]',
    'button:has-text("Submit")',
    'button:has-text("Apply")',
    'button:has-text("Send")',
    'button:has-text("Continue")',
    'button:has-text("Next")',
    'button:has-text("Finish")',
    'button:has-text("Complete")',
    '[role="button"]:has-text("Submit")',
    '[role="button"]:has-text("Apply")',
  ];

  for (const selector of submitSelectors) {
    const button = page.locator(selector).first();
    if (await button.isVisible({ timeout: SUBMIT_VISIBILITY_TIMEOUT_MS })) {
      const urlBefore = page.url();
      await button.click();

      // Wait for URL change — safer than networkidle (no WebSocket/SSE hangs)
      try {
        await page.waitForURL(url => url.toString() !== urlBefore, { timeout: SUBMIT_NAV_TIMEOUT_MS });
      } catch {
        // URL didn't change — form may have submitted without navigation (e.g., JS SPA)
        log.warn({ url: page.url() }, 'No navigation detected after submit');
      }
      return true;
    }
  }
  return false;
}
