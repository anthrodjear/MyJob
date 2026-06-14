# UI Design Tokens

> Color palette, typography, spacing, and visual tokens for the MyJob dashboard.
> All values are defined as CSS custom properties in `globals.css` and consumed via Tailwind.

---

## CSS Custom Properties

Add to `src/app/globals.css` under `:root`:

```css
:root {
  /* ── Brand ── */
  --color-primary: #2563eb;
  --color-primary-hover: #1d4ed8;
  --color-primary-light: #eff6ff;
  --color-primary-dark: #1e40af;

  /* ── Semantic: Success / Auto-Apply ── */
  --color-success: #16a34a;
  --color-success-hover: #15803d;
  --color-success-light: #f0fdf4;
  --color-success-dark: #166534;

  /* ── Semantic: Warning / Review Required ── */
  --color-warning: #d97706;
  --color-warning-hover: #b45309;
  --color-warning-light: #fffbeb;
  --color-warning-dark: #92400e;

  /* ── Semantic: Danger / Rejected ── */
  --color-danger: #dc2626;
  --color-danger-hover: #b91c1c;
  --color-danger-light: #fef2f2;
  --color-danger-dark: #991b1b;

  /* ── Semantic: Info / Applied ── */
  --color-info: #0891b2;
  --color-info-hover: #0e7490;
  --color-info-light: #ecfeff;
  --color-info-dark: #155e75;

  /* ── Neutrals ── */
  --color-bg: #ffffff;
  --color-bg-secondary: #f8fafc;
  --color-bg-tertiary: #f1f5f9;
  --color-surface: #ffffff;
  --color-surface-hover: #f8fafc;
  --color-border: #e2e8f0;
  --color-border-strong: #cbd5e1;
  --color-text-primary: #0f172a;
  --color-text-secondary: #475569;
  --color-text-tertiary: #94a3b8;
  --color-text-inverse: #ffffff;

  /* ── Score Colors ── */
  --color-score-high: #16a34a;      /* ≥ 80% match */
  --color-score-high-bg: #f0fdf4;
  --color-score-mid: #d97706;       /* 50-79% match */
  --color-score-mid-bg: #fffbeb;
  --color-score-low: #dc2626;       /* < 50% match */
  --color-score-low-bg: #fef2f2;

  /* ── Source Badge Colors ── */
  --color-source-indeed: #2164f3;
  --color-source-greenhouse: #24a800;
  --color-source-lever: #428bca;
  --color-source-remoteok: #00b67a;
  --color-source-linkedin: #0a66c2;
  --color-source-custom: #64748b;

  /* ── Shadows ── */
  --shadow-xs: 0 1px 2px 0 rgb(0 0 0 / 0.05);
  --shadow-sm: 0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1);
  --shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1);
  --shadow-lg: 0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1);
  --shadow-xl: 0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1);

  /* ── Typography ── */
  --font-sans: var(--font-geist-sans), ui-sans-serif, system-ui, sans-serif;
  --font-mono: var(--font-geist-mono), ui-monospace, monospace;
  --font-size-xs: 0.75rem;     /* 12px */
  --font-size-sm: 0.875rem;    /* 14px — body default */
  --font-size-base: 1rem;      /* 16px */
  --font-size-lg: 1.125rem;    /* 18px */
  --font-size-xl: 1.25rem;     /* 20px */
  --font-size-2xl: 1.5rem;     /* 24px */
  --font-size-3xl: 1.875rem;   /* 30px */
  --font-size-4xl: 2.25rem;    /* 36px */
  --line-height-tight: 1.25;
  --line-height-normal: 1.5;
  --line-height-relaxed: 1.75;

  /* ── Spacing (4px base) ── */
  --space-0: 0;
  --space-1: 0.25rem;     /* 4px */
  --space-2: 0.5rem;      /* 8px */
  --space-3: 0.75rem;     /* 12px */
  --space-4: 1rem;        /* 16px */
  --space-5: 1.25rem;     /* 20px */
  --space-6: 1.5rem;      /* 24px */
  --space-8: 2rem;        /* 32px */
  --space-10: 2.5rem;     /* 40px */
  --space-12: 3rem;       /* 48px */
  --space-16: 4rem;       /* 64px */
  --space-20: 5rem;       /* 80px */

  /* ── Border Radius ── */
  --radius-sm: 0.25rem;   /* 4px  — badges, small elements */
  --radius-md: 0.375rem;  /* 6px  — inputs, buttons */
  --radius-lg: 0.5rem;    /* 8px  — cards */
  --radius-xl: 0.75rem;   /* 12px — modals, large cards */
  --radius-2xl: 1rem;     /* 16px — dashboard cards */
  --radius-full: 9999px;  /* pills, avatars */

  /* ── Z-Index ── */
  --z-base: 0;
  --z-dropdown: 50;
  --z-sticky: 100;
  --z-modal-backdrop: 200;
  --z-modal: 300;
  --z-toast: 400;
  --z-tooltip: 500;
}
```

---

## Tailwind Theme Extension

Map CSS variables to Tailwind classes in `globals.css` using `@theme inline`:

```css
@theme inline {
  --color-background: var(--color-bg);
  --color-foreground: var(--color-text-primary);

  /* Brand */
  --color-primary: var(--color-primary);
  --color-primary-hover: var(--color-primary-hover);
  --color-primary-light: var(--color-primary-light);
  --color-primary-dark: var(--color-primary-dark);

  /* Semantic */
  --color-success: var(--color-success);
  --color-success-light: var(--color-success-light);
  --color-success-dark: var(--color-success-dark);
  --color-warning: var(--color-warning);
  --color-warning-light: var(--color-warning-light);
  --color-warning-dark: var(--color-warning-dark);
  --color-danger: var(--color-danger);
  --color-danger-light: var(--color-danger-light);
  --color-danger-dark: var(--color-danger-dark);
  --color-info: var(--color-info);
  --color-info-light: var(--color-info-light);
  --color-info-dark: var(--color-info-dark);

  /* Surfaces */
  --color-bg-secondary: var(--color-bg-secondary);
  --color-surface: var(--color-surface);
  --color-surface-hover: var(--color-surface-hover);
  --color-border: var(--color-border);
  --color-border-strong: var(--color-border-strong);
  --color-text-secondary: var(--color-text-secondary);
  --color-text-tertiary: var(--color-text-tertiary);
  --color-text-inverse: var(--color-text-inverse);

  /* Typography */
  --font-sans: var(--font-sans);
  --font-mono: var(--font-mono);
}
```

---

## Color Palette Reference

### Primary (Actions, Links, Focus)

| Token | Hex | Usage |
|---|---|---|
| `--color-primary` | `#2563eb` | Primary buttons, links, active states |
| `--color-primary-hover` | `#1d4ed8` | Button hover, link hover |
| `--color-primary-light` | `#eff6ff` | Selected row background, light badges |
| `--color-primary-dark` | `#1e40af` | High-contrast text on light bg |

### Success (Auto-Apply, Approved, High Match)

| Token | Hex | Usage |
|---|---|---|
| `--color-success` | `#16a34a` | Approved status, auto-apply enabled |
| `--color-success-hover` | `#15803d` | Success button hover |
| `--color-success-light` | `#f0fdf4` | Success badge background, score ≥ 80% |
| `--color-success-dark` | `#166534` | Success badge text |

### Warning (Review Required, Medium Match)

| Token | Hex | Usage |
|---|---|---|
| `--color-warning` | `#d97706` | Pending review, needs attention |
| `--color-warning-hover` | `#b45309` | Warning button hover |
| `--color-warning-light` | `#fffbeb` | Warning badge background, score 50-79% |
| `--color-warning-dark` | `#92400e` | Warning badge text |

### Danger (Rejected, Errors, Low Match)

| Token | Hex | Usage |
|---|---|---|
| `--color-danger` | `#dc2626` | Rejected status, errors, delete actions |
| `--color-danger-hover` | `#b91c1c` | Danger button hover |
| `--color-danger-light` | `#fef2f2` | Error background, score < 50% |
| `--color-danger-dark` | `#991b1b` | Danger badge text |

### Info (Applied, In Progress)

| Token | Hex | Usage |
|---|---|---|
| `--color-info` | `#0891b2` | "Applied" status, in-progress indicators |
| `--color-info-hover` | `#0e7490` | Info button hover |
| `--color-info-light` | `#ecfeff` | Info badge background |
| `--color-info-dark` | `#155e75` | Info badge text |

### Neutrals (Text, Borders, Backgrounds)

| Token | Hex | Usage |
|---|---|---|
| `--color-bg` | `#ffffff` | Page background |
| `--color-bg-secondary` | `#f8fafc` | Sidebar, secondary panels |
| `--color-bg-tertiary` | `#f1f5f9` | Hover backgrounds, dividers |
| `--color-surface` | `#ffffff` | Cards, modals, dropdowns |
| `--color-border` | `#e2e8f0` | Default borders, dividers |
| `--color-border-strong` | `#cbd5e1` | Input borders, emphasis |
| `--color-text-primary` | `#0f172a` | Headings, body text |
| `--color-text-secondary` | `#475569` | Descriptions, labels |
| `--color-text-tertiary` | `#94a3b8` | Placeholders, timestamps |

### Source Badge Colors

| Token | Hex | Source |
|---|---|---|
| `--color-source-indeed` | `#2164f3` | Indeed |
| `--color-source-greenhouse` | `#24a800` | Greenhouse |
| `--color-source-lever` | `#428bca` | Lever |
| `--color-source-remoteok` | `#00b67a` | RemoteOK |
| `--color-source-linkedin` | `#0a66c2` | LinkedIn |
| `--color-source-custom` | `#64748b` | Custom/user-added |

---

## Typography Scale

| Level | Size | Weight | Line Height | Usage |
|---|---|---|---|---|
| `text-xs` | 12px | 400 | 1.5 | Timestamps, labels, micro text |
| `text-sm` | 14px | 400 | 1.5 | **Body default**, descriptions |
| `text-base` | 16px | 400 | 1.5 | Emphasized body text |
| `text-lg` | 18px | 500 | 1.5 | Card titles, section headers |
| `text-xl` | 20px | 600 | 1.25 | Page titles |
| `text-2xl` | 24px | 600 | 1.25 | Dashboard headings |
| `text-3xl` | 30px | 700 | 1.25 | KPI numbers, hero stats |
| `text-4xl` | 36px | 700 | 1.25 | Dashboard page title |

### Font Weights

| Token | Weight | Usage |
|---|---|---|
| `font-normal` | 400 | Body text, descriptions |
| `font-medium` | 500 | Labels, emphasis |
| `font-semibold` | 600 | Headings, card titles |
| `font-bold` | 700 | KPI numbers, page titles |

### Monospace (Data & Numbers)

- Use `font-mono tabular-nums` for: match scores, timestamps, IDs, statistics.
- Ensures number columns align properly in tables.

---

## Spacing Scale

Based on 4px grid. Use Tailwind spacing classes directly:

| Token | Value | Common Usage |
|---|---|---|
| `1` | 4px | Tight padding, icon gaps |
| `2` | 8px | Inline element gaps, small padding |
| `3` | 12px | Default padding in compact areas |
| `4` | 16px | Standard padding, card padding |
| `5` | 20px | Medium spacing |
| `6` | 24px | Card padding, section gaps |
| `8` | 32px | Large section gaps |
| `10` | 40px | Section separation |
| `12` | 48px | Page-level spacing |
| `16` | 64px | Major section breaks |

---

## Border Radius

| Token | Value | Usage |
|---|---|---|
| `rounded-sm` | 4px | Badges, small elements, tags |
| `rounded-md` | 6px | Buttons, inputs, form fields |
| `rounded-lg` | 8px | Cards, panels |
| `rounded-xl` | 12px | Modals, large cards, dropdowns |
| `rounded-2xl` | 16px | Dashboard stat cards |
| `rounded-full` | 9999px | Avatars, status dots, pills |

---

## Shadows

| Token | Usage |
|---|---|
| `shadow-xs` | Subtle depth on flat cards |
| `shadow-sm` | Default card shadow, dropdowns |
| `shadow-md` | Hover state on cards, sticky elements |
| `shadow-lg` | Modals, popovers |
| `shadow-xl` | Toast notifications, floating elements |

---

## Dark Mode Tokens

Add `@media (prefers-color-scheme: dark)` overrides:

```css
@media (prefers-color-scheme: dark) {
  :root {
    --color-bg: #0a0a0a;
    --color-bg-secondary: #111111;
    --color-bg-tertiary: #1a1a1a;
    --color-surface: #141414;
    --color-surface-hover: #1a1a1a;
    --color-border: #262626;
    --color-border-strong: #404040;
    --color-text-primary: #fafafa;
    --color-text-secondary: #a1a1a1;
    --color-text-tertiary: #737373;

    /* Semantic colors: keep same hue, adjust lightness */
    --color-primary-light: #172554;
    --color-success-light: #14532d;
    --color-warning-light: #451a03;
    --color-danger-light: #450a0a;
    --color-info-light: #164e63;
  }
}
```

---

## Quick Reference: Semantic Status Mapping

| Status | Badge Color | Background | Text |
|---|---|---|---|
| Discovered | `--color-text-tertiary` | `--color-bg-tertiary` | `--color-text-secondary` |
| Applied | `--color-info` | `--color-info-light` | `--color-info-dark` |
| Review Required | `--color-warning` | `--color-warning-light` | `--color-warning-dark` |
| Approved / Auto-Applied | `--color-success` | `--color-success-light` | `--color-success-dark` |
| Rejected | `--color-danger` | `--color-danger-light` | `--color-danger-dark` |
| Interview Scheduled | `--color-primary` | `--color-primary-light` | `--color-primary-dark` |
| Offer | `--color-success` | `--color-success-light` | `--color-success-dark` |

---

**Status:** Token system defined. Ready to be applied to `globals.css`.
**Last updated:** 2026-06-14
