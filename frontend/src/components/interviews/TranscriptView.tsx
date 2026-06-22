/**
 * TranscriptView — displays interview transcript entries.
 *
 * Renders speaker-labeled turns with timestamps in a scrollable log.
 * Uses role="log" for screen reader live region support.
 *
 * @example
 *   <TranscriptView entries={session.transcript} />
 */

"use client";

import { cn, formatDate } from "@/lib/utils";
import type { TranscriptEntry, TranscriptSpeaker } from "@/lib/types/interviews";

const SPEAKER_STYLES: Record<TranscriptSpeaker, string> = {
  candidate: "bg-blue-100 text-blue-900 dark:bg-blue-900/30 dark:text-blue-200",
  ai: "bg-purple-100 text-purple-900 dark:bg-purple-900/30 dark:text-purple-200",
  system: "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400",
};

const SPEAKER_LABELS: Record<TranscriptSpeaker, string> = {
  candidate: "You",
  ai: "AI Interviewer",
  system: "System",
};

interface TranscriptViewProps {
  /** Array of transcript entries to display. */
  entries: TranscriptEntry[];
}

export function TranscriptView({ entries }: TranscriptViewProps) {
  if (entries.length === 0) {
    return (
      <p className="text-sm text-muted-foreground italic">
        No transcript entries yet.
      </p>
    );
  }

  return (
    <div className="space-y-4" role="log" aria-label="Interview transcript">
      {entries.map((entry) => (
        <div key={entry.id} className="flex gap-3">
          <div className="shrink-0 pt-0.5">
            <span
              className={cn(
                "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
                SPEAKER_STYLES[entry.speaker]
              )}
            >
              {SPEAKER_LABELS[entry.speaker]}
            </span>
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-sm">{entry.content}</p>
            <time
              dateTime={entry.timestamp}
              className="mt-1 block text-xs text-muted-foreground"
            >
              {formatDate(entry.timestamp)}
            </time>
          </div>
        </div>
      ))}
    </div>
  );
}
