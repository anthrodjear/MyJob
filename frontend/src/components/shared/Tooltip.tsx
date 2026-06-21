/**
 * Tooltip — hover-triggered informational popup.
 *
 * Shows a small popup above the trigger element on hover/focus.
 * Uses `role="tooltip"` for screen reader semantics.
 *
 * @example
 *   <Tooltip content="Auto-apply threshold">
 *     <Button variant="ghost" size="sm">?</Button>
 *   </Tooltip>
 */

"use client";

import { type ReactNode, useId, useState } from "react";
import { cn } from "@/lib/utils";

interface TooltipProps {
  /** Tooltip text content. */
  content: string;
  /** The trigger element that shows the tooltip on hover. */
  children: ReactNode;
  /** Additional CSS classes for the wrapper. */
  className?: string;
}

/**
 * Tooltip — hover-triggered informational popup.
 *
 * Accessibility:
 * - `role="tooltip"` on the popup element
 * - Trigger element should have accessible name (tooltip supplements, not replaces)
 * - Shown on hover and focus (native browser behavior for focus)
 * - Hidden when not hovered (no persistent state)
 *
 * Implementation:
 * - Uses `useState` for show/hide
 * - Positioned above the trigger with `absolute bottom-full`
 * - Arrow indicator pointing down to the trigger
 * - Dark background (`bg-text-primary`) with light text (`text-text-inverse`)
 */
export function Tooltip({ content, children, className }: TooltipProps) {
  const [show, setShow] = useState(false);
  const tooltipId = useId();

  return (
    <div
      className={cn("relative inline-block", className)}
      onMouseEnter={() => setShow(true)}
      onMouseLeave={() => setShow(false)}
      onFocus={() => setShow(true)}
      onBlur={() => setShow(false)}
    >
      <div aria-describedby={show ? tooltipId : undefined}>{children}</div>
      {show && (
        <div
          id={tooltipId}
          className="absolute bottom-full left-1/2 z-[--z-tooltip] mb-2 -translate-x-1/2 rounded-md bg-text-primary px-2 py-1 text-xs text-text-inverse whitespace-nowrap"
          role="tooltip"
        >
          {content}
          <div className="absolute top-full left-1/2 -translate-x-1/2 border-4 border-transparent border-t-text-primary" />
        </div>
      )}
    </div>
  );
}
