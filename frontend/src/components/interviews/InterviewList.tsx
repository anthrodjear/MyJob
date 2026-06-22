/**
 * InterviewList — scrollable list of interview session cards.
 *
 * Handles loading, empty, and list states.
 * Renders interviews in a vertical stack using InterviewCard for individual display.
 *
 * @example
 *   <InterviewList interviews={sessions} isLoading={false} onSelect={handleSelect} />
 */

"use client";

import { Mic } from "lucide-react";
import { InterviewCard } from "./InterviewCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { Skeleton } from "@/components/shared/LoadingSkeleton";
import type { InterviewSession } from "@/lib/types/interviews";

interface InterviewListProps {
  /** Array of interview sessions to display. */
  interviews: InterviewSession[];
  /** Whether the list is currently loading. */
  isLoading?: boolean;
  /** Callback when an interview card is clicked. */
  onSelect?: (interview: InterviewSession) => void;
}

/** Skeleton placeholder for a single interview card during loading. */
function SkeletonCard() {
  return (
    <div className="rounded-lg border p-4">
      <div className="flex items-start gap-4">
        <Skeleton className="h-5 w-5 shrink-0 rounded" />
        <div className="flex-1 space-y-2">
          <div className="flex gap-2">
            <Skeleton className="h-5 w-16 rounded-full" />
            <Skeleton className="h-4 w-12" />
          </div>
          <div className="flex gap-4">
            <Skeleton className="h-3 w-20" />
            <Skeleton className="h-3 w-24" />
          </div>
        </div>
      </div>
    </div>
  );
}

export function InterviewList({ interviews, isLoading, onSelect }: InterviewListProps) {
  if (isLoading) {
    return (
      <div className="space-y-3" aria-busy="true" aria-label="Loading interviews">
        {Array.from({ length: 4 }, (_, i) => (
          <SkeletonCard key={i} />
        ))}
      </div>
    );
  }

  if (interviews.length === 0) {
    return (
      <EmptyState
        icon={<Mic className="h-12 w-12" />}
        title="No interviews yet"
        description="Start an interview session from an application."
      />
    );
  }

  return (
    <div className="space-y-3" role="list" aria-label="Interview sessions">
      {interviews.map((interview) => (
        <div key={interview.id} role="listitem">
          <InterviewCard interview={interview} onClick={() => onSelect?.(interview)} />
        </div>
      ))}
    </div>
  );
}
