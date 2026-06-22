/**
 * Application detail page — fetches single application and renders detail view.
 *
 * Client Component. Uses useApplication and useApplicationTimeline hooks.
 */

"use client";

import { use } from "react";
import { useRouter } from "next/navigation";
import { useApplication, useApplicationTimeline, useUpdateApplicationStatus, useUpdateApplicationNotes } from "@/hooks/useApplications";
import { ApplicationDetail } from "@/components/applications/ApplicationDetail";
import { Button } from "@/components/shared/Button";
import type { ApplicationStatus } from "@/lib/types/applications";

interface ApplicationDetailPageProps {
  params: Promise<{ id: string }>;
}

export default function ApplicationDetailPage({ params }: ApplicationDetailPageProps) {
  const { id } = use(params);
  const router = useRouter();

  const { data: application, isLoading, error } = useApplication(id);
  const { data: timelineData } = useApplicationTimeline(id);
  const updateStatusMutation = useUpdateApplicationStatus();
  const updateNotesMutation = useUpdateApplicationNotes();

  const handleStatusChange = (applicationId: string, status: ApplicationStatus) => {
    updateStatusMutation.mutate({ id: applicationId, status });
  };

  const handleNotesSave = (applicationId: string, notes: string) => {
    updateNotesMutation.mutate({ id: applicationId, notes });
  };

  if (isLoading) {
    return (
      <div className="py-12 text-center">
        <p className="text-sm text-text-tertiary">Loading application...</p>
      </div>
    );
  }

  if (error || !application) {
    return (
      <div className="py-12 text-center" aria-live="assertive">
        <p className="text-sm text-danger">Failed to load application.</p>
        <Button variant="secondary" size="sm" className="mt-4" onClick={() => router.push("/dashboard/applications")}>
          Back to Applications
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Button
        variant="ghost"
        size="sm"
        onClick={() => router.push("/dashboard/applications")}
        aria-label="Back to applications list"
      >
        ← Back to Applications
      </Button>
      <ApplicationDetail
        application={application}
        timeline={timelineData?.events}
        onStatusChange={handleStatusChange}
        onNotesSave={handleNotesSave}
        isUpdating={updateStatusMutation.isPending || updateNotesMutation.isPending}
      />
    </div>
  );
}
