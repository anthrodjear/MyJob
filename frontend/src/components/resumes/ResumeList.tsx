/**
 * ResumeList — renders a list of resumes with empty state.
 *
 * Uses ResumeCard for each item. Shows EmptyState when no resumes exist.
 *
 * @example
 *   <ResumeList resumes={resumes} />
 */

"use client";

import { FileText } from "lucide-react";
import type { Resume } from "@/lib/types/resumes";
import { ResumeCard } from "./ResumeCard";
import { EmptyState } from "@/components/shared/EmptyState";

interface ResumeListProps {
  resumes: Resume[];
}

export function ResumeList({ resumes }: ResumeListProps) {
  if (resumes.length === 0) {
    return (
      <EmptyState
        icon={<FileText className="h-12 w-12" />}
        title="No resumes yet"
        description="Create your first resume to get started with AI-powered job applications."
        action={{
          label: "Create Resume",
          onClick: () => { window.location.href = "/dashboard/resumes/new"; },
        }}
      />
    );
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {resumes.map((resume) => (
        <ResumeCard key={resume.id} resume={resume} />
      ))}
    </div>
  );
}
