import { Page } from 'playwright';
import { logger } from '../utils/logger.js';

const log = logger.child({ component: 'FieldDetector' });

/**
 * A selectable option within a `<select>`, radio group, or checkbox group.
 */
export interface FieldOption {
  /** The underlying `value` attribute. */
  value: string;
  /** Human-readable display text. */
  label: string;
}

/**
 * Represents a detected form field on a page.
 *
 * Radio buttons sharing the same `name` are collapsed into a single logical
 * field with `type: 'radio'` and populated `options`.  Checkboxes sharing
 * the same `name` are collapsed similarly with `type: 'checkbox'`.
 */
export interface FormField {
  /** Element `id` attribute (may be empty). */
  id: string;
  /** Element `name` attribute (may be empty). */
  name: string;
  /** HTML input type (text, email, select, textarea, radio, checkbox, etc.). */
  type: string;
  /** Resolved human-readable label (from `<label>`, aria-label, or placeholder). */
  label: string;
  /** Placeholder text, if any. */
  placeholder: string;
  /** `autocomplete` attribute value, if set. */
  autocomplete: string;
  /** Whether the field is required. */
  required: boolean;
  /** Whether the field is disabled. */
  disabled: boolean;
  /** Whether the field is read-only. */
  readonly: boolean;
  /** Options for `<select>`, radio, or checkbox groups; empty for other types. */
  options: FieldOption[];
  /** CSS selector uniquely targeting this field. */
  selector: string;
  /** Whether the field was found inside a Shadow DOM root. */
  inShadowRoot: boolean;
}

// ── Pure functions (testable without browser) ─────────────────────────────

/** Strip asterisks, colons, and collapse whitespace. */
export function normalizeLabel(raw: string | null | undefined): string {
  return (raw ?? '').replace(/[*:]/g, '').replace(/\s+/g, ' ').trim();
}

/** Escape a string for use inside a quoted CSS attribute value. */
export function escapeAttr(s: string): string {
  return s.replace(/\\/g, '\\\\').replace(/"/g, '\\"').replace(/\[/g, '\\[').replace(/\]/g, '\\]').replace(/,/g, '\\,').replace(/:/g, '\\:');
}

function safeCSSescape(s: string): string {
  try {
    return CSS.escape(s);
  } catch {
    return escapeAttr(s);
  }
}

function getLabel(
  el: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement,
  root: Document | ShadowRoot,
): string {
  // label[for=id] — look in the same root (light DOM or shadow root)
  if (el.id) {
    const label = root.querySelector(`label[for="${safeCSSescape(el.id)}"]`);
    if (label) return normalizeLabel(label.textContent);
  }
  // parent <label>
  const parentLabel = el.closest('label');
  if (parentLabel) return normalizeLabel(parentLabel.textContent);
  // aria-labelledby
  const labelledBy = el.getAttribute('aria-labelledby');
  if (labelledBy) {
    const parts = labelledBy.split(/\s+/).map(id => {
      // Search both light DOM and shadow roots
      const ref = root.getElementById?.(id) ?? document.getElementById(id);
      return ref?.textContent ?? '';
    }).filter(Boolean);
    if (parts.length > 0) return normalizeLabel(parts.join(' '));
  }
  // aria-label
  const ariaLabel = el.getAttribute('aria-label');
  if (ariaLabel) return ariaLabel.trim();
  // placeholder
  if ('placeholder' in el && el.placeholder) return el.placeholder.trim();
  // preceding sibling text (only text-level elements)
  const prev = el.previousElementSibling;
  if (prev && /^(SPAN|LABEL|EM|STRONG|B|I|SMALL|ABBR)$/.test(prev.tagName)) {
    const text = prev.textContent?.trim() ?? '';
    if (text) return normalizeLabel(text);
  }
  return '';
}

/**
 * Generate a stable CSS selector for an element.
 * Prefers stable attributes: data-testid, data-qa, data-cy, name, id.
 * Falls back to ancestor CSS path with :nth-child only as last resort.
 */
export function generateSelector(el: Element): string {
  // data-testid (Playwright / React Testing Library)
  const testId = el.getAttribute('data-testid');
  if (testId) return `[data-testid="${escapeAttr(testId)}"]`;
  // data-qa (QA automation)
  const qa = el.getAttribute('data-qa');
  if (qa) return `[data-qa="${escapeAttr(qa)}"]`;
  // data-cy (Cypress)
  const cy = el.getAttribute('data-cy');
  if (cy) return `[data-cy="${escapeAttr(cy)}"]`;
  // id
  if (el.id) return `#${safeCSSescape(el.id)}`;
  // name
  const name = el.getAttribute('name');
  if (name) return `[name="${escapeAttr(name)}"]`;
  // aria-label
  const ariaLabel = el.getAttribute('aria-label');
  if (ariaLabel) return `[aria-label="${escapeAttr(ariaLabel)}"]`;
  // Fallback: ancestor path with :nth-child
  const parts: string[] = [];
  let current: Element | null = el;
  while (current && current !== document.body) {
    let part = current.tagName.toLowerCase();
    if (current.id) {
      part += `#${safeCSSescape(current.id)}`;
      parts.unshift(part);
      break;
    }
    if (current.className && typeof current.className === 'string') {
      const classes = current.className.split(' ')
        .filter(c => c && !c.startsWith('css-'))
        .slice(0, 2);
      if (classes.length > 0) part += '.' + classes.map(c => safeCSSescape(c)).join('.');
    }
    const siblings = current.parentElement ? Array.from(current.parentElement.children) : [];
    const index = siblings.indexOf(current);
    if (index >= 0) part += `:nth-child(${index + 1})`;
    parts.unshift(part);
    current = current.parentElement;
  }
  return parts.join(' > ');
}

function isVisible(el: Element): boolean {
  const htmlEl = el as HTMLElement;
  const style = window.getComputedStyle(htmlEl);
  if (style.display === 'none' || style.visibility === 'hidden') return false;
  // offsetParent is null for fixed-position elements — only reject if not fixed
  if (htmlEl.offsetParent === null && style.position !== 'fixed') return false;
  return true;
}

// Recursively collects input elements from light DOM and shadow roots.
function collectInputs(
  root: Document | ShadowRoot,
  results: Array<{ element: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement; root: Document | ShadowRoot }>,
): void {
  const INPUT_SELECTOR = 'input, textarea, select';
  const inputs = Array.from(root.querySelectorAll<Element>(INPUT_SELECTOR));
  for (const el of inputs) {
    results.push({ element: el as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement, root });
    // If this element hosts a shadow root, recurse into it
    if (el.shadowRoot) {
      collectInputs(el.shadowRoot, results);
    }
  }
  // Also check non-input custom elements that might have shadow roots
  // Only check custom elements (tag name contains '-') to avoid O(n²)
  const allEls = Array.from(root.querySelectorAll<HTMLElement>('*:not(' + INPUT_SELECTOR + ')'));
  for (const el of allEls) {
    if (el.shadowRoot && el.tagName.includes('-')) {
      collectInputs(el.shadowRoot, results);
    }
  }
}

interface RawField {
  element: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement;
  root: Document | ShadowRoot;
  id: string;
  name: string;
  type: string;
  label: string;
  placeholder: string;
  autocomplete: string;
  required: boolean;
  disabled: boolean;
  readonly: boolean;
}

const SKIP_TYPES = new Set(['hidden', 'submit', 'button', 'reset', 'image']);

/**
 * Detect all visible, fillable form fields on the current page.
 *
 * Traverses Shadow DOM roots recursively to detect custom elements
 * (e.g. `<my-input>`) that wrap standard `<input>` elements.
 *
 * Fields are returned in DOM order.  Non-fillable fields (hidden, submit,
 * button, disabled, invisible) are filtered out.  Radio and checkbox inputs
 * sharing the same `name` are grouped into a single logical field.
 *
 * @param page - Playwright Page instance (must be navigated to target URL).
 * @returns Array of detected form fields.
 */
export async function detectFormFields(page: Page): Promise<FormField[]> {
  const fields = await page.evaluate(() => {
    // Collect from light DOM + shadow roots
    const allInputs: Array<{ element: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement; root: Document | ShadowRoot }> = [];
    collectInputs(document, allInputs);

    // Filter to fillable, visible fields
    const rawFields: RawField[] = allInputs
      .filter(({ element: el }) => {
        if (SKIP_TYPES.has(el.type)) return false;
        if (!isVisible(el)) return false;
        return true;
      })
      .map(({ element: el, root }) => ({
        element: el,
        root,
        id: el.id || '',
        name: el.getAttribute('name') || '',
        type: (() => {
          if (el.tagName === 'SELECT') return 'select';
          if (el.tagName === 'TEXTAREA') return 'textarea';
          return el.type || 'text';
        })(),
        label: getLabel(el, root),
        placeholder: 'placeholder' in el ? (el.placeholder || '') : '',
        autocomplete: el.getAttribute('autocomplete') || '',
        required: el.required,
        disabled: el.disabled,
        readonly: 'readOnly' in el ? (el as HTMLInputElement).readOnly : false,
      }));

    // ── Group radio buttons by name ─────────────────────────────────
    const radioGroups = new Map<string, RawField[]>();
    const nonGrouped: RawField[] = [];

    for (const rf of rawFields) {
      if (rf.type === 'radio' && rf.name) {
        const existing = radioGroups.get(rf.name);
        if (existing) {
          existing.push(rf);
        } else {
          radioGroups.set(rf.name, [rf]);
        }
      } else {
        nonGrouped.push(rf);
      }
    }

    // ── Group checkboxes by name ────────────────────────────────────
    const checkboxGroups = new Map<string, RawField[]>();
    const finalFields: RawField[] = [];

    for (const rf of nonGrouped) {
      if (rf.type === 'checkbox' && rf.name) {
        const existing = checkboxGroups.get(rf.name);
        if (existing) {
          existing.push(rf);
        } else {
          checkboxGroups.set(rf.name, [rf]);
        }
      } else {
        finalFields.push(rf);
      }
    }

    // ── Build FormField results ─────────────────────────────────────
    const result: FormField[] = [];

    // Non-grouped fields
    for (const rf of finalFields) {
      const inShadow = rf.root !== document;
      const field: FormField = {
        id: rf.id,
        name: rf.name,
        type: rf.type,
        label: rf.label,
        placeholder: rf.placeholder,
        autocomplete: rf.autocomplete,
        required: rf.required,
        disabled: rf.disabled,
        readonly: rf.readonly,
        options: rf.element.tagName === 'SELECT'
          ? Array.from((rf.element as HTMLSelectElement).options).map(o => ({
              value: o.value,
              label: o.textContent?.trim() || o.value,
            }))
          : [],
        selector: generateSelector(rf.element),
        inShadowRoot: inShadow,
      };
      result.push(field);
    }

    // Radio groups → single logical field
    for (const [name, radios] of radioGroups) {
      const first = radios[0];
      result.push({
        id: '', // Group has no single id
        name,
        type: 'radio',
        label: first.label,
        placeholder: first.placeholder,
        autocomplete: first.autocomplete,
        required: first.required,
        disabled: first.disabled,
        readonly: first.readonly,
        options: radios.map(r => {
          const labelEl = r.root.querySelector(`label[for="${safeCSSescape(r.element.id)}"]`);
          return {
            value: r.element.value || '',
            label: r.element.getAttribute('aria-label')
              || labelEl?.textContent?.trim()
              || r.label
              || r.element.value
              || '',
          };
        }),
        selector: generateSelector(first.element),
        inShadowRoot: first.root !== document,
      });
    }

    // Checkbox groups → single logical field
    for (const [name, checkboxes] of checkboxGroups) {
      const first = checkboxes[0];
      result.push({
        id: '', // Group has no single id
        name,
        type: 'checkbox',
        label: first.label,
        placeholder: first.placeholder,
        autocomplete: first.autocomplete,
        required: first.required,
        disabled: first.disabled,
        readonly: first.readonly,
        options: checkboxes.map(c => {
          const labelEl = c.root.querySelector(`label[for="${safeCSSescape(c.element.id)}"]`);
          return {
            value: c.element.value || '',
            label: c.element.getAttribute('aria-label')
              || labelEl?.textContent?.trim()
              || c.label
              || c.element.value
              || '',
          };
        }),
        selector: generateSelector(first.element),
        inShadowRoot: first.root !== document,
      });
    }

    return result;
  }) as FormField[];

  log.debug({ fieldsFound: fields.length }, 'Form field detection complete');
  return fields;
}