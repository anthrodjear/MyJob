/**
 * InterviewList — scrollable list of interview session cards.
 *
 * Handles loading, empty, and list states.
 * Renders interviews in a vertical stack using InterviewCard for individual display.
 * Uses SkeletonWrapper to enforce min/max display times and prevent pop-ins.
 *
 * @example
 *   <InterviewList interviews={sessions} isLoading={false} onSelect={handleSelect} />
 */

"use client";

import { Mic } from "lucide-react";
import { InterviewCard } from "./InterviewCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { InterviewCardSkeleton, SkeletonWrapper } from "@/components/shared/LoadingSkeleton";
import type { InterviewSession } from "@/lib/types/interviews";

interface InterviewListProps {
  /** Array of interview sessions to display. */
  interviews: InterviewSession[];
  /** Whether the list is currently loading. */
  isLoading?: boolean;
  /** Callback when an interview card is clicked. */
  onSelect?: (interview: InterviewSession) => void;
}

/** Skeleton placeholder matching the list layout. */
function InterviewListSkeleton() {
  return (
    <div aria-busy="true" aria-label="Loading interviews">
      <span className="sr-only" aria-live="polite">Loading interviews…</span>
      <div className="space-y-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <InterviewCardSkeleton key={i} />
        ))}
      </div>
    </div>
  );
}

export function InterviewList({ interviews, isLoading = false, onSelect }: InterviewListProps) {
  // Use SkeletonWrapper to enforce min/max display times and prevent pop-ins
  return (
    <SkeletonWrapper
      isLoading={isLoading}
      skeleton={<InterviewListSkeleton />}
      minDisplayMs={300}
      maxDisplayMs={5000}
      ariaLiveRegion="Interviews loaded"
    >
      <div className="space-y-3" role="list" aria-label="Interview sessions">
        {/* Empty state */}
        {interviews.length === 0 && !isLoading && (
          <EmptyState
            icon={<Mic className="h-12 w-12" />}
            title="No interviews yet"
            description="Start an interview session from an application."
          />
        )}

        {/* Interviews list */}
        {interviews.length > 0 && (
          <>
            {interviews.map((interview) => (
              <div key={interview.id} role="listitem">
                <InterviewCard interview={interview} onClick={() => onSelect?.(interview)} />
              </div>
            ))}
          </>
        )}
      </div>
    </SkeletonWrapper>
  );
}
