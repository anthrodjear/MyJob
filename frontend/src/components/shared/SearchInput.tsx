/**
 * SearchInput — debounced text search field with icon.
 *
 * Fires `onSearch` after the user stops typing (default: 300ms debounce).
 * Includes a search icon and consistent styling.
 *
 * @example
 *   <SearchInput onSearch={handleSearch} placeholder="Search jobs..." />
 *   <SearchInput onSearch={handleSearch} debounceMs={500} />
 */

"use client";

import { type InputHTMLAttributes, useCallback, useEffect, useRef } from "react";
import { cn } from "@/lib/utils";
import { Search } from "lucide-react";

interface SearchInputProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, "onChange"> {
  /** Called with the debounced input value. */
  onSearch: (value: string) => void;
  /** Debounce delay in milliseconds. Default: 300. */
  debounceMs?: number;
}

/**
 * SearchInput — debounced text search field.
 *
 * Accessibility:
 * - Search icon is decorative (`aria-hidden` via Lucide)
 * - Input inherits all standard HTML input attributes
 * - Placeholder text provides context
 *
 * Implementation:
 * - Uses a ref-based debounce (no timer cleanup needed on unmount)
 * - `onSearch` is only called when the value actually changes
 * - Input value is uncontrolled (matches native behavior)
 */
export function SearchInput({
  onSearch,
  debounceMs = 300,
  className,
  ...props
}: SearchInputProps) {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value;

      // Clear any pending debounce
      if (timerRef.current != null) {
        clearTimeout(timerRef.current);
      }

      // Set new debounce
      timerRef.current = setTimeout(() => {
        onSearch(value);
      }, debounceMs);
    },
    [onSearch, debounceMs],
  );

  // Cleanup pending timer on unmount
  useEffect(() => {
    return () => {
      if (timerRef.current != null) clearTimeout(timerRef.current);
    };
  }, []);

  return (
    <div className={cn("relative", className)}>
      <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-tertiary" aria-hidden="true" />
      <input
        type="text"
        className={cn(
          "w-full rounded-md border border-border bg-surface py-2 pl-9 pr-3 text-sm",
          "placeholder:text-text-tertiary",
          "focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary",
        )}
        onChange={handleChange}
        {...props}
      />
    </div>
  );
}
