"use client";

/**
 * Button component — primary interactive element.
 *
 * Provides 4 variants (primary, secondary, ghost, danger) and 3 sizes.
 * Includes loading state with accessible spinner and `aria-busy`.
 *
 * Uses React 19 ref-as-prop pattern (no forwardRef needed).
 *
 * @example
 *   <Button variant="primary" size="md">Submit</Button>
 *   <Button variant="ghost" loading={saving}>Save</Button>
 *   <Button variant="danger" onClick={handleDelete}>Delete</Button>
 */

import { type ComponentPropsWithRef, type ReactNode } from "react";
import { cn } from "@/lib/utils";

/** Button visual variant — maps to semantic color tokens. */
type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";

/** Button size — controls padding and font size. */
type ButtonSize = "sm" | "md" | "lg";

interface ButtonProps extends ComponentPropsWithRef<"button"> {
  /** Visual variant. Default: "primary". */
  variant?: ButtonVariant;
  /** Size. Default: "md". */
  size?: ButtonSize;
  /** Button label. */
  children: ReactNode;
  /** Shows a spinner and disables the button. Default: false. */
  loading?: boolean;
  /** Accessible label for the loading state. Default: "Loading…". */
  loadingText?: string;
}

/**
 * Variant-to-Tailwind class mapping.
 * Uses design tokens from globals.css (@theme inline).
 */
const variantStyles: Record<ButtonVariant, string> = {
  primary:
    "bg-primary text-text-inverse hover:bg-primary-hover focus-visible:ring-primary",
  secondary:
    "bg-bg-tertiary text-text-primary border border-border hover:bg-border focus-visible:ring-primary",
  ghost:
    "bg-transparent text-text-secondary hover:bg-bg-tertiary hover:text-text-primary focus-visible:ring-primary",
  danger:
    "bg-danger text-text-inverse hover:bg-danger-hover focus-visible:ring-danger",
};

/**
 * Size-to-Tailwind class mapping.
 */
const sizeStyles: Record<ButtonSize, string> = {
  sm: "px-3 py-1.5 text-sm",
  md: "px-4 py-2 text-sm",
  lg: "px-6 py-3 text-base",
};

/**
 * Accessible loading spinner (inline SVG).
 * Decorative — hidden from screen readers via `aria-hidden="true"`.
 */
function Spinner() {
  return (
    <svg
      className="mr-2 h-4 w-4 animate-spin"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      aria-hidden="true"
    >
      <circle
        className="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        strokeWidth="4"
      />
      <path
        className="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
      />
    </svg>
  );
}

/**
 * Button — primary interactive element.
 *
 * Accessibility:
 * - `disabled` when loading or explicitly disabled
 * - `aria-busy={loading}` for screen reader loading announcement
 * - `focus-visible:ring` for keyboard navigation
 * - `disabled:pointer-events-none disabled:opacity-50` visual disabled state
 *
 * Usage:
 * - Always provide explicit `variant` for clarity
 * - Use `loading` for async operations (form submits, API calls)
 * - Use `loadingText` for accessible loading context ("Saving…", "Deleting…")
 */
export function Button({
  variant = "primary",
  size = "md",
  className,
  children,
  loading = false,
  loadingText,
  disabled,
  ref,
  ...props
}: ButtonProps) {
  return (
    <button
      ref={ref}
      type="button"
      className={cn(
        // Base styles
        "inline-flex items-center justify-center rounded-md font-medium",
        "transition-colors duration-150",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2",
        "disabled:pointer-events-none disabled:opacity-50",
        // Variant and size
        variantStyles[variant],
        sizeStyles[size],
        className,
      )}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      aria-label={
        loading
          ? (loadingText ?? props["aria-label"] ?? "Loading")
          : props["aria-label"]
      }
      {...props}
    >
      {loading && <Spinner />}
      {children}
    </button>
  );
}


