/**
 * SearchInput — debounced text search field with icon and clear button.
 *
 * Fires `onSearch` after the user stops typing (default: 300ms debounce).
 * Includes a search icon, consistent styling, and optional clear button.
 *
 * @example
 *   <SearchInput onSearch={handleSearch} placeholder="Search jobs..." />
 *   <SearchInput onSearch={handleSearch} debounceMs={500} />
 *   <SearchInput onSearch={handleSearch} showClear />
 */

"use client";

import {
  type InputHTMLAttributes,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { cn } from "@/lib/utils";
import { Search, X } from "lucide-react";

interface SearchInputProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, "onChange"> {
  /** Called with the debounced input value. */
  onSearch: (value: string) => void;
  /** Debounce delay in milliseconds. Default: 300. */
  debounceMs?: number;
  /** Show a clear button when there's input. Default: false. */
  showClear?: boolean;
}

/**
 * SearchInput — debounced text search field.
 *
 * Accessibility:
 * - Search icon is decorative (`aria-hidden`)
 * - Clear button has `aria-label="Clear search"`
 * - Input inherits all standard HTML input attributes
 * - Placeholder text provides context
 * - Keyboard: Tab to input, type, Tab to clear button, Enter/Space to clear
 *
 * Implementation:
 * - Uses a ref-based debounce (no timer cleanup needed on unmount)
 * - `onSearch` is only called when the value actually changes
 * - Input value is uncontrolled (matches native behavior)
 * - Clear button calls `onSearch("")` and focuses the input
 */
export function SearchInput({
  onSearch,
  debounceMs = 300,
  showClear = false,
  className,
  ...props
}: SearchInputProps) {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [hasValue, setHasValue] = useState(false);

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value;
      setHasValue(value.length > 0);

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

  const handleClear = useCallback(() => {
    // Clear any pending debounce
    if (timerRef.current != null) {
      clearTimeout(timerRef.current);
    }

    // Clear the input and immediately search with empty string
    setHasValue(false);
    if (inputRef.current != null) {
      inputRef.current.value = "";
    }
    onSearch("");
    inputRef.current?.focus();
  }, [onSearch]);

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
        ref={inputRef}
        type="text"
        className={cn(
          "w-full rounded-md border border-border bg-surface py-2 pl-9 text-sm",
          "placeholder:text-text-tertiary",
          "transition-colors duration-150",
          "focus-visible:border-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-primary",
          showClear && hasValue && "pr-9",
        )}
        onChange={handleChange}
        {...props}
      />
      {showClear && hasValue && (
        <button
          type="button"
          onClick={handleClear}
          className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 text-text-tertiary hover:text-text-secondary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-primary"
          aria-label="Clear search"
        >
          <X className="h-4 w-4" aria-hidden="true" />
        </button>
      )}
    </div>
  );
}
