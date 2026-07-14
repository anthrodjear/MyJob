/**
 * Loading skeleton — animated placeholder for content being loaded.
 *
 * Provides Skeleton (base), CardSkeleton, TableRowSkeleton, JobCardSkeleton,
 * and content-specific skeletons for consistent loading states.
 *
 * No `"use client"` — pure presentational, works in Server Components.
 *
 * @example
 *   <Skeleton className="h-4 w-3/4" />
 *   <CardSkeleton />
 *   <JobCardSkeleton />
 *   <SkeletonWrapper minDisplayMs={300} maxDisplayMs={5000} isLoading={isLoading}>
 *     <RealContent />
 *   </SkeletonWrapper>
 */

import { useEffect, useRef, useState } from "react";
import { cn } from "@/lib/utils";

interface SkeletonProps {
  /** Additional CSS classes (width, height, etc.). */
  className?: string;
}

/**
 * Base skeleton block — animated pulse or shimmer placeholder.
 * Use specific width/height classes to match the expected content shape.
 *
 * Accessibility:
 * - `aria-hidden="true"` — decorative, hidden from screen readers
 * - `animate-pulse` / `animate-shimmer` — subtle animation indicates loading state
 * - Respects `prefers-reduced-motion`
 */
export function Skeleton({ className, variant = "pulse" }: SkeletonProps & { variant?: "pulse" | "shimmer" }) {
  return (
    <div
      className={cn(
        "rounded-md bg-bg-tertiary",
        variant === "shimmer" ? "animate-shimmer" : "motion-safe:animate-pulse",
        className
      )}
      aria-hidden="true"
    />
  );
}

/**
 * Card-shaped skeleton — matches Card component dimensions.
 * Shows 3 placeholder lines of varying widths for a realistic layout.
 */
export function CardSkeleton({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "rounded-xl border border-border bg-surface p-6 shadow-sm",
        className,
      )}
    >
      <Skeleton className="mb-3 h-4 w-3/4" />
      <Skeleton className="mb-2 h-3 w-1/2" />
      <Skeleton className="h-3 w-2/3" />
    </div>
  );
}

/**
 * Table row skeleton — matches DataTable row dimensions.
 * Renders N placeholder cells (default: 5) for a single table row.
 *
 * @param columns - Number of columns to render. Default: 5.
 */
export function TableRowSkeleton({ columns = 5 }: { columns?: number }) {
  return (
    <tr>
      {Array.from({ length: columns }).map((_, i) => (
        <td key={i} className="px-4 py-3">
          <Skeleton className="h-4 w-full" />
        </td>
      ))}
    </tr>
  );
}

/**
 * Job card skeleton — matches JobCard component layout exactly.
 * Includes: company logo placeholder, title line, company/location line,
 * salary range, tags, and action buttons area.
 */
export function JobCardSkeleton({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "rounded-xl border border-border bg-surface p-5 shadow-sm transition-shadow hover:shadow-md",
        className,
      )}
    >
      {/* Header: logo + title + company */}
      <div className="flex items-start gap-3 mb-3">
        <Skeleton className="h-12 w-12 rounded-lg shrink-0" variant="shimmer" />
        <div className="flex-1 min-w-0">
          <Skeleton className="h-5 w-3/4 mb-1.5" />
          <Skeleton className="h-4 w-1/2" variant="shimmer" />
          <Skeleton className="h-3 w-3/4 mt-1" variant="shimmer" />
        </div>
      </div>

      {/* Meta: location, remote, salary */}
      <div className="flex flex-wrap items-center gap-3 text-sm text-text-secondary mb-3">
        <Skeleton className="h-3 w-24" variant="shimmer" />
        <Skeleton className="h-3 w-20" variant="shimmer" />
        <Skeleton className="h-3 w-28" variant="shimmer" />
      </div>

      {/* Tags */}
      <div className="flex flex-wrap gap-2 mb-4">
        <Skeleton className="h-6 w-20 rounded-full" variant="shimmer" />
        <Skeleton className="h-6 w-24 rounded-full" variant="shimmer" />
        <Skeleton className="h-6 w-16 rounded-full" variant="shimmer" />
      </div>

      {/* Action buttons */}
      <div className="flex items-center justify-between pt-3 border-t border-border">
        <Skeleton className="h-10 w-24 rounded-md" variant="shimmer" />
        <div className="flex gap-2">
          <Skeleton className="h-10 w-20 rounded-md" variant="shimmer" />
          <Skeleton className="h-10 w-20 rounded-md" variant="shimmer" />
        </div>
      </div>
    </div>
  );
}

/**
 * Application card skeleton — matches ApplicationCard component layout.
 * Includes: job title, company, status badge, date, actions.
 */
export function ApplicationCardSkeleton({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "rounded-xl border border-border bg-surface p-5 shadow-sm transition-shadow hover:shadow-md",
        className,
      )}
    >
      {/* Header: logo + title + company */}
      <div className="flex items-start gap-3 mb-3">
        <Skeleton className="h-12 w-12 rounded-lg shrink-0" variant="shimmer" />
        <div className="flex-1 min-w-0">
          <Skeleton className="h-5 w-3/4 mb-1.5" />
          <Skeleton className="h-4 w-1/2" variant="shimmer" />
          <Skeleton className="h-3 w-3/4 mt-1" variant="shimmer" />
        </div>
      </div>

      {/* Meta: status, applied date, tier */}
      <div className="flex flex-wrap items-center gap-3 text-sm text-text-secondary mb-3">
        <Skeleton className="h-6 w-24 rounded-full" variant="shimmer" />
        <Skeleton className="h-3 w-24" variant="shimmer" />
        <Skeleton className="h-3 w-20" variant="shimmer" />
      </div>

      {/* Score / progress */}
      <div className="mb-4">
        <Skeleton className="h-2 w-full rounded-full mb-1" variant="shimmer" />
        <div className="flex items-center justify-between text-xs text-text-tertiary">
          <Skeleton className="h-3 w-16" variant="shimmer" />
          <Skeleton className="h-3 w-12" variant="shimmer" />
        </div>
      </div>

      {/* Action buttons */}
      <div className="flex items-center justify-between pt-3 border-t border-border">
        <Skeleton className="h-10 w-24 rounded-md" variant="shimmer" />
        <div className="flex gap-2">
          <Skeleton className="h-10 w-20 rounded-md" variant="shimmer" />
          <Skeleton className="h-10 w-20 rounded-md" variant="shimmer" />
        </div>
      </div>
    </div>
  );
}

/**
 * Resume card skeleton — matches ResumeCard component layout.
 * Includes: template preview placeholder, name, specialization, version, actions.
 */
export function ResumeCardSkeleton({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "rounded-xl border border-border bg-surface p-5 shadow-sm transition-shadow hover:shadow-md",
        className,
      )}
    >
      {/* Template preview placeholder */}
      <Skeleton className="aspect-video w-full rounded-lg mb-4" variant="shimmer" />

      {/* Name + specialization */}
      <Skeleton className="h-5 w-3/4 mb-1.5" />
      <Skeleton className="h-4 w-1/2" variant="shimmer" />

      {/* Template path + version */}
      <div className="flex flex-wrap items-center gap-3 text-sm text-text-secondary mb-4">
        <Skeleton className="h-3 w-24" variant="shimmer" />
        <Skeleton className="h-3 w-16" variant="shimmer" />
      </div>

      {/* Action buttons */}
      <div className="flex items-center justify-between pt-3 border-t border-border">
        <Skeleton className="h-10 w-24 rounded-md" variant="shimmer" />
        <div className="flex gap-2">
          <Skeleton className="h-10 w-20 rounded-md" variant="shimmer" />
          <Skeleton className="h-10 w-20 rounded-md" variant="shimmer" />
        </div>
      </div>
    </div>
  );
}

/**
 * Cover letter card skeleton — matches CoverLetterCard component layout.
 * Includes: company logo, job title, company name, generation date, actions.
 */
export function CoverLetterCardSkeleton({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "rounded-xl border border-border bg-surface p-5 shadow-sm transition-shadow hover:shadow-md",
        className,
      )}
    >
      {/* Header: logo + job title + company */}
      <div className="flex items-start gap-3 mb-3">
        <Skeleton className="h-12 w-12 rounded-lg shrink-0" variant="shimmer" />
        <div className="flex-1 min-w-0">
          <Skeleton className="h-5 w-3/4 mb-1.5" />
          <Skeleton className="h-4 w-1/2" variant="shimmer" />
          <Skeleton className="h-3 w-3/4 mt-1" variant="shimmer" />
        </div>
      </div>

      {/* Meta: generated date, word count, model */}
      <div className="flex flex-wrap items-center gap-3 text-sm text-text-secondary mb-3">
        <Skeleton className="h-3 w-24" variant="shimmer" />
        <Skeleton className="h-3 w-20" variant="shimmer" />
        <Skeleton className="h-3 w-28" variant="shimmer" />
      </div>

      {/* Preview lines */}
      <div className="space-y-2 mb-4">
        <Skeleton className="h-3 w-full" variant="shimmer" />
        <Skeleton className="h-3 w-5/6" variant="shimmer" />
        <Skeleton className="h-3 w-2/3" variant="shimmer" />
      </div>

      {/* Action buttons */}
      <div className="flex items-center justify-between pt-3 border-t border-border">
        <Skeleton className="h-10 w-24 rounded-md" variant="shimmer" />
        <div className="flex gap-2">
          <Skeleton className="h-10 w-20 rounded-md" variant="shimmer" />
          <Skeleton className="h-10 w-20 rounded-md" variant="shimmer" />
        </div>
      </div>
    </div>
  );
}

/**
 * Approval card skeleton — matches ApprovalCard component layout.
 * Includes: company logo, job snapshot, resume/cover letter previews, approve/reject actions.
 */
export function ApprovalCardSkeleton({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "rounded-xl border border-border bg-surface p-5 shadow-sm transition-shadow hover:shadow-md",
        className,
      )}
    >
      {/* Header: logo + job title + company */}
      <div className="flex items-start gap-3 mb-3">
        <Skeleton className="h-12 w-12 rounded-lg shrink-0" variant="shimmer" />
        <div className="flex-1 min-w-0">
          <Skeleton className="h-5 w-3/4 mb-1.5" />
          <Skeleton className="h-4 w-1/2" variant="shimmer" />
          <Skeleton className="h-3 w-3/4 mt-1" variant="shimmer" />
        </div>
      </div>

      {/* Job snapshot preview */}
      <div className="rounded-lg border border-border bg-bg-secondary p-3 mb-4">
        <Skeleton className="h-4 w-3/4 mb-1.5" />
        <Skeleton className="h-3 w-1/2" variant="shimmer" />
        <Skeleton className="h-3 w-2/3 mt-1" variant="shimmer" />
      </div>

      {/* Preview tabs: resume / cover letter */}
      <div className="flex gap-2 mb-4">
        <Skeleton className="h-8 w-24 rounded-md" variant="shimmer" />
        <Skeleton className="h-8 w-28 rounded-md" variant="shimmer" />
      </div>

      {/* Action buttons: approve / reject */}
      <div className="flex items-center justify-between pt-3 border-t border-border">
        <div className="flex gap-2">
          <Skeleton className="h-10 w-24 rounded-md" variant="shimmer" />
          <Skeleton className="h-10 w-24 rounded-md" variant="shimmer" />
        </div>
        <Skeleton className="h-10 w-20 rounded-md" variant="shimmer" />
      </div>
    </div>
  );
}

/**
 * Email card skeleton — matches EmailCard component layout.
 */
export function EmailCardSkeleton({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "rounded-xl border border-border bg-surface p-4 shadow-sm transition-shadow hover:shadow-md",
        className,
      )}
    >
      <div className="flex items-start justify-between gap-3 mb-2">
        <div className="flex-1 min-w-0">
          <Skeleton className="h-5 w-3/4 mb-1" />
          <Skeleton className="h-3 w-1/2" variant="shimmer" />
        </div>
        <Skeleton className="h-6 w-20 rounded-full shrink-0" variant="shimmer" />
      </div>
      <Skeleton className="h-3 w-full mb-1" variant="shimmer" />
      <Skeleton className="h-3 w-3/4" variant="shimmer" />
      <div className="flex items-center justify-between pt-3 mt-3 border-t border-border">
        <Skeleton className="h-8 w-20 rounded-md" variant="shimmer" />
        <div className="flex gap-2">
          <Skeleton className="h-8 w-16 rounded-md" variant="shimmer" />
          <Skeleton className="h-8 w-16 rounded-md" variant="shimmer" />
        </div>
      </div>
    </div>
  );
}

/**
 * Interview card skeleton — matches InterviewCard component layout.
 */
export function InterviewCardSkeleton({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "rounded-xl border border-border bg-surface p-5 shadow-sm transition-shadow hover:shadow-md",
        className,
      )}
    >
      <div className="flex items-start gap-3 mb-3">
        <Skeleton className="h-12 w-12 rounded-lg shrink-0" variant="shimmer" />
        <div className="flex-1 min-w-0">
          <Skeleton className="h-5 w-3/4 mb-1.5" />
          <Skeleton className="h-4 w-1/2" variant="shimmer" />
          <Skeleton className="h-3 w-3/4 mt-1" variant="shimmer" />
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3 text-sm text-text-secondary mb-3">
        <Skeleton className="h-3 w-24" variant="shimmer" />
        <Skeleton className="h-3 w-20" variant="shimmer" />
        <Skeleton className="h-3 w-28" variant="shimmer" />
      </div>

      <div className="flex items-center justify-between pt-3 border-t border-border">
        <Skeleton className="h-10 w-24 rounded-md" variant="shimmer" />
        <div className="flex gap-2">
          <Skeleton className="h-10 w-20 rounded-md" variant="shimmer" />
          <Skeleton className="h-10 w-20 rounded-md" variant="shimmer" />
        </div>
      </div>
    </div>
  );
}

/**
 * Task card skeleton — matches TaskCard component layout.
 * Includes: status icon + type badge, status badge, metadata (attempts, priority, duration, time), pulse bar.
 */
export function TaskCardSkeleton({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "rounded-lg border border-border bg-bg-secondary p-4 transition-colors",
        className,
      )}
    >
      {/* Header: status icon + type badge, status badge */}
      <div className="flex items-start justify-between gap-3 mb-3">
        <div className="flex items-center gap-2 min-w-0">
          <Skeleton className="h-4 w-4 shrink-0 rounded" variant="shimmer" />
          <Skeleton className="h-5 w-24 rounded-full shrink-0" variant="shimmer" />
        </div>
        <Skeleton className="h-5 w-20 rounded-full shrink-0" variant="shimmer" />
      </div>

      {/* Metadata: attempts/priority, duration/time */}
      <div className="flex items-center justify-between text-xs text-text-tertiary mb-3">
        <div className="flex items-center gap-3">
          <Skeleton className="h-3 w-24" variant="shimmer" />
          <Skeleton className="h-3 w-16" variant="shimmer" />
        </div>
        <div className="flex items-center gap-3">
          <Skeleton className="h-3 w-16" variant="shimmer" />
          <Skeleton className="h-3 w-24" variant="shimmer" />
        </div>
      </div>

      {/* Pulse bar for active tasks */}
      <div className="h-1 overflow-hidden rounded-full bg-bg-tertiary">
        <div className="h-full w-full animate-pulse rounded-full bg-info/50" />
      </div>
    </div>
  );
}

/**
 * SkeletonWrapper — wraps real content and shows skeleton while loading.
 * Enforces minimum display time (prevents flash-of-content) and maximum
 * display time (prevents infinite shimmer).
 *
 * @param isLoading - Whether to show skeleton
 * @param children - Real content to render when not loading
 * @param skeleton - Skeleton component to show while loading
 * @param minDisplayMs - Minimum time to show skeleton (prevents flash). Default: 300ms
 * @param maxDisplayMs - Maximum time to show skeleton (prevents infinite shimmer). Default: 5000ms
 * @param onTimeout - Called when maxDisplayMs exceeded (show error/fallback). Can return ReactNode/string for custom fallback.
 * @param ariaLiveRegion - ARIA live region text to announce when loading completes
 */
export function SkeletonWrapper({
  isLoading,
  children,
  skeleton,
  minDisplayMs = 300,
  maxDisplayMs = 5000,
  onTimeout,
  ariaLiveRegion = "Content loaded",
}: {
  isLoading: boolean;
  children: React.ReactNode;
  skeleton: React.ReactNode;
  minDisplayMs?: number;
  maxDisplayMs?: number;
  onTimeout?: () => React.ReactElement | string;
  ariaLiveRegion?: string;
}) {
  const [showSkeleton, setShowSkeleton] = useState(true);
  const [hasTimedOut, setHasTimedOut] = useState(false);
  const startTimeRef = useRef<number>(0);
  const minTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const maxTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const liveRegionRef = useRef<HTMLDivElement>(null);

  // Announce content loaded to screen readers
  useEffect(() => {
    if (!isLoading && liveRegionRef.current) {
      liveRegionRef.current.textContent = ariaLiveRegion;
      // Clear after announcement
      setTimeout(() => {
        if (liveRegionRef.current) {
          liveRegionRef.current.textContent = "";
        }
      }, 1000);
    }
  }, [isLoading, ariaLiveRegion]);

  useEffect(() => {
    if (isLoading) {
      startTimeRef.current = Date.now();

      // Minimum display time - prevent flash of content
      minTimerRef.current = setTimeout(() => {
        // Allow showing real content after min time if loading is done
      }, minDisplayMs);

      // Maximum display time - prevent infinite shimmer
      maxTimerRef.current = setTimeout(() => {
        setHasTimedOut(true);
        onTimeout?.();
      }, maxDisplayMs);
    } else {
      // Loading finished - wait for min time if needed
      const elapsed = Date.now() - startTimeRef.current;
      const remaining = Math.max(0, minDisplayMs - elapsed);

      if (remaining > 0) {
        minTimerRef.current = setTimeout(() => {
          setShowSkeleton(false);
        }, remaining);
      } else {
        setShowSkeleton(false);
      }
    }

    return () => {
      if (minTimerRef.current) clearTimeout(minTimerRef.current);
      if (maxTimerRef.current) clearTimeout(maxTimerRef.current);
    };
  }, [isLoading, minDisplayMs, maxDisplayMs, onTimeout]);

  if (hasTimedOut && onTimeout) {
    return <div role="alert" aria-live="assertive">{onTimeout()}</div>;
  }

  return (
    <>
      {showSkeleton ? skeleton : children}
      <div
        ref={liveRegionRef}
        role="status"
        aria-live="polite"
        aria-atomic="true"
        className="sr-only"
      />
    </>
  );
}

/**
 * Directional shimmer animation keyframes (CSS-in-JS style via Tailwind).
 * Add to your global CSS or use via `animate-shimmer` class:
 *
 * @keyframes shimmer {
 *   0% { background-position: -200% 0; }
 *   100% { background-position: 200% 0; }
 * }
 * .animate-shimmer {
 *   background: linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%);
 *   background-size: 200% 100%;
 *   animation: shimmer 1.5s infinite;
 * }
 * @media (prefers-reduced-motion: reduce) {
 *   .animate-shimmer { animation: none; background: #e0e0e0; }
 * }
 */