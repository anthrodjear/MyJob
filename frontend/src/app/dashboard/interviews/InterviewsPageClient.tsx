/**
 * InterviewsPageClient — client-side interviews page with status filter.
 *
 * URL-synced filter state via useSearchParams + router.push.
 * Single router.push for multi-param changes to avoid stale snapshot bugs.
 *
 * @example
 *   /dashboard/interviews?status=active&limit=50
 */

"use client";

import { useState, useCallback, useMemo } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Mic } from "lucide-react";
import { useInterviews } from "@/hooks/useInterviews";
import { InterviewList } from "@/components/interviews/InterviewList";
import { InterviewDetail } from "@/components/interviews/InterviewDetail";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { InterviewSession, InterviewStatus } from "@/lib/types/interviews";

const STATUS_OPTIONS: { value: InterviewStatus; label: string }[] = [
  { value: "pending", label: "Pending" },
  { value: "starting", label: "Starting" },
  { value: "active", label: "Active" },
  { value: "completed", label: "Completed" },
  { value: "failed", label: "Failed" },
  { value: "cancelled", label: "Cancelled" },
];

export function InterviewsPageClient() {
  const router = useRouter();
  const searchParams = useSearchParams();

  const status = searchParams.get("status") as InterviewStatus | null;
  const limit = Number(searchParams.get("limit") || "50");
  const offset = Number(searchParams.get("offset") || "0");

  const [selectedInterview, setSelectedInterview] = useState<InterviewSession | null>(null);

  const { data, isLoading } = useInterviews({
    status: status ?? undefined,
    limit,
    offset,
  });

  const interviews = useMemo(() => data?.interviews ?? [], [data]);

  const handleSelect = useCallback((interview: InterviewSession) => {
    setSelectedInterview(interview);
  }, []);

  const handleBack = useCallback(() => {
    setSelectedInterview(null);
  }, []);

  if (selectedInterview) {
    return (
      <div className="mx-auto max-w-4xl">
        <InterviewDetail interview={selectedInterview} onBack={handleBack} />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Interviews</h1>
          <p className="text-muted-foreground">
            Voice interview sessions and transcripts.
          </p>
        </div>
        <Mic className="h-8 w-8 text-muted-foreground" aria-hidden="true" />
      </div>

      <div className="flex flex-col gap-4 sm:flex-row">
        <Select
          value={status ?? "all"}
          onValueChange={(value) => {
            const params = new URLSearchParams(searchParams.toString());
            if (value === "all") {
              params.delete("status");
            } else {
              params.set("status", value);
            }
            params.delete("offset");
            router.push(`/dashboard/interviews?${params.toString()}`, {
              scroll: false,
            });
          }}
        >
          <SelectTrigger className="w-full sm:w-48" aria-label="Filter by status">
            <SelectValue placeholder="All statuses" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All statuses</SelectItem>
            {STATUS_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div aria-live="polite" aria-atomic="true" aria-busy={isLoading}>
        {data?.total != null && (
          <p className="text-sm text-muted-foreground mb-4">
            {data.total} {data.total === 1 ? "session" : "sessions"}
          </p>
        )}
      </div>

      <InterviewList
        interviews={interviews}
        isLoading={isLoading}
        onSelect={handleSelect}
      />
    </div>
  );
}
