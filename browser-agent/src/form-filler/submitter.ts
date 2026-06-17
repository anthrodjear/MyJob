import { Page } from 'playwright';
import { access } from 'node:fs/promises';
import { FormField } from './detector.js';
import { FieldMapping } from './fields.js';
import { logger } from '../utils/logger.js';

// ── Constants ──────────────────────────────────────────────────────

const FIELD_WAIT_TIMEOUT_MS = 5_000;
const SUBMIT_VISIBILITY_TIMEOUT_MS = 1_000;
const SUBMIT_NAV_TIMEOUT_MS = 10_000;

// ── CSS selector escaping (Node.js has no CSS.escape) ─────────────

/**
 * Escape a string for use as a CSS ID selector.
 * Handles leading digits (invalid CSS identifiers) and special characters.
 */
function escapeId(value: string): string {
  let result = '';
  for (const ch of value) {
    if (ch >= '0' && ch <= '9' && result === '') {
      // CSS identifiers can't start with a digit — use hex escape
      result += '\\' + ch.charCodeAt(0).toString(16) + ' ';
    } else if (/[^\w-]/.test(ch)) {
      result += '\\' + ch;
    } else {
      result += ch;
    }
  }
  return result;
}

function escapeAttr(value: string): string {
  return value.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
}

/**
 * Build a CSS selector for a form field, preferring `selector` → `#id` → `[name=]`.
 */
function buildSelector(field: FormField): string {
  if (field.selector) return field.selector;
  if (field.id) return `#${escapeId(field.id)}`;
  if (field.name) return `[name="${escapeAttr(field.name)}"]`;
  return '';
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
 * Fill all mapped form fields on the page and optionally upload a resume.
 *
 * @param page - Playwright page with the form loaded.
 * @param fields - Detected form fields (from `detectFormFields`).
 * @param mappings - Field mappings (from `mapFieldsToCandidateData`).
 * @param resumePath - Optional file path to a resume PDF for file upload fields.
 * @returns Result with filled count, errors, and optional screenshot.
 */
export async function fillForm(
  page: Page,
  fields: FormField[],
  mappings: FieldMapping[],
  resumePath?: string,
): Promise<FillResult> {
  const errors: string[] = [];
  let filledCount = 0;

  // Create a map for quick lookup
  const mappingMap = new Map(mappings.map(m => [m.field_id, m]));

  for (const field of fields) {
    const mapping = mappingMap.get(field.id) || mappingMap.get(field.selector);
    if (!mapping || !mapping.value) continue;

    try {
      await fillField(page, field, mapping.value);
      filledCount++;
    } catch (error) {
      const msg = `Field ${field.label} (${field.id}): ${error instanceof Error ? error.message : String(error)}`;
      errors.push(msg);
      logger.warn(
        { fieldId: field.id, fieldType: field.type, selector: field.selector, error },
        'Failed to fill field',
      );
    }
  }

  // Handle file uploads (resume)
  if (resumePath) {
    try {
      await uploadResume(page, fields, resumePath);
    } catch (error) {
      errors.push(`Resume upload: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  // Take screenshot for verification — non-critical, log on failure
  let screenshot: Buffer | undefined;
  try {
    screenshot = await page.screenshot({ fullPage: true, type: 'png' });
  } catch (err) {
    logger.warn({ err }, 'Failed to capture form screenshot');
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
    // Text-like inputs — use fill()
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

    // Select dropdowns — try value, then label
    case 'select':
      try {
        await element.selectOption({ value });
      } catch (firstErr) {
        try {
          await element.selectOption({ label: value });
        } catch {
          throw new Error(
            `Option "${value}" not found by value or label in field "${field.label}": ${firstErr instanceof Error ? firstErr.message : String(firstErr)}`,
          );
        }
      }
      break;

    // Checkboxes — check or uncheck
    case 'checkbox':
      if (value.toLowerCase() === 'true' || value === '1' || value.toLowerCase() === 'yes') {
        await element.check();
      } else {
        await element.uncheck();
      }
      break;

    // Radio buttons — only select (never unselect)
    case 'radio':
      await element.check();
      break;

    // File inputs — handled separately in uploadResume
    case 'file':
      break;

    default:
      throw new Error(`Unsupported field type: ${field.type}`);
  }
}

// ── File upload ────────────────────────────────────────────────────

/**
 * Upload a resume file to all file input fields on the page.
 *
 * Verifies the file exists before attempting upload.
 * Logs a warning if `resumePath` is provided but no file inputs are found.
 */
async function uploadResume(page: Page, fields: FormField[], resumePath: string): Promise<void> {
  // Verify file exists before attempting upload
  try {
    await access(resumePath);
  } catch {
    throw new Error(`Resume file not found: ${resumePath}`);
  }

  const fileInputs = fields.filter(f => f.type === 'file');
  if (fileInputs.length === 0) {
    logger.warn({ fileInputCount: 0, resumePath }, 'resumePath provided but no file inputs found on page');
    return;
  }
  for (const field of fileInputs) {
    const selector = buildSelector(field);
    if (!selector) {
      logger.warn({ fieldId: field.id, fieldLabel: field.label }, 'File input has no selector, skipping upload');
      continue;
    }
    await page.locator(selector).first().setInputFiles(resumePath);
    break; // upload to first matching file input only
  }
}

// ── Submit ─────────────────────────────────────────────────────────

/**
 * Find and click the submit button on the form.
 *
 * Tries common submit selectors: `button[type=submit]`, `input[type=submit]`,
 * and buttons with text like "Submit", "Apply", "Send", "Continue", "Next".
 *
 * After clicking, waits for navigation or URL change instead of `networkidle`
 * to avoid hangs on pages with persistent connections (analytics, WebSockets).
 *
 * @returns `true` if a submit button was found and clicked, `false` otherwise.
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
        await page.waitForURL(url => url.href !== urlBefore, { timeout: SUBMIT_NAV_TIMEOUT_MS });
      } catch {
        // URL didn't change — form may have submitted without navigation (e.g., JS SPA)
        logger.warn({ url: page.url() }, 'No navigation detected after submit');
      }
      return true;
    }
  }
  return false;
}
