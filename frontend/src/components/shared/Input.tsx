/**
 * Input — reusable text input with label, error, and optional icon.
 *
 * Provides consistent styling across all forms.
 * Supports password visibility toggle, left/right icons, and error states.
 *
 * No `"use client"` — pure presentational, works in Server Components.
 * (Only becomes client when used with useState in consuming components.)
 *
 * @example
 *   <Input label="Password" type="password" error="Required" />
 *   <Input label="Search" leftIcon={<Search />} placeholder="Type to search…" />
 *   <Input label="API Key" rightIcon={<Key />} />
 */

import { type ComponentPropsWithRef, type ReactNode, useId } from "react";
import { cn } from "@/lib/utils";

interface InputProps extends Omit<ComponentPropsWithRef<"input">, "size"> {
  /** Visible label above the input. */
  label?: string;
  /** Error message displayed below the input. Also styles the border red. */
  error?: string;
  /** Helper text displayed below the input (hidden when error is present). */
  helperText?: string;
  /** Icon rendered inside the left edge of the input. */
  leftIcon?: ReactNode;
  /** Icon rendered inside the right edge of the input (before password toggle). */
  rightIcon?: ReactNode;
  /** Input size. Default: "md". */
  size?: "sm" | "md" | "lg";
}

/**
 * Size-to-Tailwind class mapping for the input element.
 */
const sizeStyles: Record<"sm" | "md" | "lg", string> = {
  sm: "px-3 py-1.5 text-sm",
  md: "px-3 py-2 text-sm",
  lg: "px-4 py-2.5 text-base",
};

/**
 * Input — reusable text input with label, error, and icon support.
 *
 * Accessibility:
 * - `<label>` linked via `htmlFor` / `id`
 * - `aria-invalid` when error is present
 * - `aria-describedby` points to error or helper text
 * - `aria-required` when `required` prop is set
 */
export function Input({
  label,
  error,
  helperText,
  leftIcon,
  rightIcon,
  size = "md",
  className,
  id: idProp,
  required,
  disabled,
  ref,
  ...props
}: InputProps) {
  const autoId = useId();
  const id = idProp ?? autoId;
  const errorId = `${id}-error`;
  const helperId = `${id}-helper`;
  const hasError = error != null && error !== "";

  return (
    <div className={cn("w-full", className)}>
      {/* Label */}
      {label != null && label !== "" && (
        <label
          htmlFor={id}
          className="mb-1 block text-sm font-medium text-text-primary"
        >
          {label}
          {required && (
            <span className="ml-0.5 text-danger" aria-hidden="true">
              *
            </span>
          )}
        </label>
      )}

      {/* Input wrapper */}
      <div className="relative">
        {/* Left icon */}
        {leftIcon != null && (
          <div className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-text-tertiary" aria-hidden="true">
            {leftIcon}
          </div>
        )}

        <input
          ref={ref}
          id={id}
          required={required}
          disabled={disabled}
          aria-invalid={hasError || undefined}
          aria-describedby={
            hasError
              ? errorId
              : helperText != null && helperText !== ""
                ? helperId
                : undefined
          }
          aria-required={required || undefined}
          className={cn(
            // Base
            "block w-full rounded-md border bg-surface text-text-primary placeholder:text-text-tertiary",
            "transition-colors duration-150",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2",
            "disabled:cursor-not-allowed disabled:opacity-50",
            // Size
            sizeStyles[size],
            // Left icon padding
            leftIcon != null && "pl-9",
            // Right icon padding (when rightIcon is present and not interactive)
            rightIcon != null && "pr-9",
            // Error state
            hasError
              ? "border-danger focus-visible:border-danger focus-visible:ring-danger"
              : "border-border focus-visible:border-primary focus-visible:ring-primary",
          )}
          {...props}
        />

        {/* Right icon */}
        {rightIcon != null && (
          <div className="absolute right-3 top-1/2 -translate-y-1/2 text-text-tertiary">
            {rightIcon}
          </div>
        )}
      </div>

      {/* Error message */}
      {hasError && (
        <p id={errorId} className="mt-1 text-sm text-danger" role="alert">
          {error}
        </p>
      )}

      {/* Helper text (hidden when error is shown) */}
      {helperText != null &&
        helperText !== "" &&
        !hasError && (
          <p id={helperId} className="mt-1 text-sm text-text-tertiary">
            {helperText}
          </p>
        )}
    </div>
  );
}
