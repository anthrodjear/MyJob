/**
 * InterviewCard — clickable interview session preview card.
 *
 * Shows session metadata (mode, provider, status, score) with active state styling.
 * Client Component — uses onClick for interactivity.
 *
 * @example
 *   <InterviewCard interview={session} onClick={() => setSelected(session)} />
 */

"use client";

import { Mic } from "lucide-react";
import { cn, formatDate } from "@/lib/utils";
import { StatusBadge } from "./StatusBadge";
import type { InterviewSession } from "@/lib/types/interviews";

interface InterviewCardProps {
  /** Interview session data to display. */
  interview: InterviewSession;
  /** Click handler for selection. */
  onClick?: () => void;
}

export function InterviewCard({ interview, onClick }: InterviewCardProps) {
  const isActive = interview.status === "active";

  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex w-full items-start gap-4 rounded-lg border p-4 text-left transition-colors",
        "hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        isActive && "border-green-200 bg-green-50/50 dark:border-green-900/50 dark:bg-green-900/10"
      )}
      aria-label={`Interview session: ${interview.mode} mode, status ${interview.status}`}
    >
      <div className="mt-0.5 shrink-0">
        <Mic
          className={cn(
            "h-5 w-5",
            isActive ? "text-green-500" : "text-muted-foreground"
          )}
          aria-hidden="true"
        />
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <StatusBadge status={interview.status} />
            <span className="text-xs text-muted-foreground capitalize">
              {interview.mode}
            </span>
          </div>
          <time
            dateTime={interview.created_at}
            className="shrink-0 text-xs text-muted-foreground"
          >
            {formatDate(interview.created_at)}
          </time>
        </div>

        <div className="mt-2 flex items-center gap-4 text-xs text-muted-foreground">
          {interview.provider && <span>{interview.provider}</span>}
          {interview.model && <span>{interview.model}</span>}
          {interview.transcript.length > 0 && (
            <span>{interview.transcript.length} turns</span>
          )}
          {interview.score != null && (
            <span className="font-medium text-foreground">
              Score: {Math.round(interview.score * 100)}%
            </span>
          )}
        </div>

        {interview.started_at && interview.ended_at && (
          <p className="mt-1 text-xs text-muted-foreground">
            Duration:{" "}
            {Math.round(
              (new Date(interview.ended_at).getTime() -
                new Date(interview.started_at).getTime()) /
                60000
            )}{" "}
            min
          </p>
        )}
      </div>
    </button>
  );
}
