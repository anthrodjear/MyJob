/**
 * InterviewDetail — full interview session view with actions.
 *
 * Shows session metadata, start/stop controls, score, and transcript.
 * Client Component — uses hooks for mutation states.
 *
 * @example
 *   <InterviewDetail interview={session} onBack={() => setSelected(null)} />
 */

"use client";

import { ArrowLeft, Play, Square } from "lucide-react";
import { formatDate } from "@/lib/utils";
import { StatusBadge } from "./StatusBadge";
import { TranscriptView } from "./TranscriptView";
import { Button } from "@/components/shared/Button";
import { useStartInterview, useStopInterview } from "@/hooks/useInterviews";
import type { InterviewSession } from "@/lib/types/interviews";

interface InterviewDetailProps {
  /** Interview session data to display. */
  interview: InterviewSession;
  /** Callback to navigate back. */
  onBack?: () => void;
}

export function InterviewDetail({ interview, onBack }: InterviewDetailProps) {
  const startInterview = useStartInterview();
  const stopInterview = useStopInterview();

  const canStart = interview.status === "pending";
  const canStop = interview.status === "active";

  return (
    <div className="space-y-6">
      {onBack && (
        <Button variant="ghost" size="sm" onClick={onBack} className="gap-1.5">
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          Back
        </Button>
      )}

      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-semibold">Interview Session</h2>
            <StatusBadge status={interview.status} />
          </div>
          <div className="mt-1 flex items-center gap-2 text-sm text-muted-foreground">
            <span className="capitalize">{interview.mode} mode</span>
            {interview.provider && (
              <>
                <span aria-hidden="true">·</span>
                <span>{interview.provider}</span>
              </>
            )}
            {interview.model && (
              <>
                <span aria-hidden="true">·</span>
                <span>{interview.model}</span>
              </>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          {canStart && (
            <Button
              size="sm"
              onClick={() => startInterview.mutate({ id: interview.id })}
              loading={startInterview.isPending}
              loadingText="Starting…"
            >
              <Play className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
              Start
            </Button>
          )}
          {canStop && (
            <Button
              size="sm"
              variant="danger"
              onClick={() => stopInterview.mutate({ id: interview.id })}
              loading={stopInterview.isPending}
              loadingText="Stopping…"
            >
              <Square className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
              Stop
            </Button>
          )}
        </div>
      </div>

      {(startInterview.isError || stopInterview.isError) && (
        <p className="text-xs text-destructive" role="alert">
          Failed to update interview. Please try again.
        </p>
      )}

      <div className="grid gap-4 sm:grid-cols-3">
        <div className="rounded-lg border p-3">
          <p className="text-xs text-muted-foreground">Started</p>
          <p className="text-sm font-medium">
            {interview.started_at ? formatDate(interview.started_at) : "—"}
          </p>
        </div>
        <div className="rounded-lg border p-3">
          <p className="text-xs text-muted-foreground">Ended</p>
          <p className="text-sm font-medium">
            {interview.ended_at ? formatDate(interview.ended_at) : "—"}
          </p>
        </div>
        <div className="rounded-lg border p-3">
          <p className="text-xs text-muted-foreground">Turns</p>
          <p className="text-sm font-medium">{interview.transcript.length}</p>
        </div>
      </div>

      {interview.score != null && (
        <div className="rounded-lg border p-4">
          <p className="text-sm text-muted-foreground">Score</p>
          <p className="text-2xl font-bold">
            {Math.round(interview.score * 100)}%
          </p>
        </div>
      )}

      <div>
        <h3 className="mb-3 text-sm font-medium">Transcript</h3>
        <TranscriptView entries={interview.transcript} />
      </div>
    </div>
  );
}
