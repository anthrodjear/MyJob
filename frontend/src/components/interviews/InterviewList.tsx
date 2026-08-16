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
  // Empty state — early return to avoid nesting role="status" inside role="list"
  if (interviews.length === 0 && !isLoading) {
    return (
      <EmptyState
        icon={<Mic className="h-12 w-12" />}
        title="No interview sessions"
        description="AI-powered interview practice sessions will appear here. Start one from an application's detail page."
        hint="Apply to jobs first — interview prep is available for each application."
        action={{ label: "View Applications", onClick: () => { window.location.href = "/dashboard/applications"; } }}
      />
    );
  }

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
        {interviews.map((interview) => (
          <div key={interview.id} role="listitem">
            <InterviewCard interview={interview} onClick={() => onSelect?.(interview)} />
          </div>
        ))}
      </div>
    </SkeletonWrapper>
  );
}
