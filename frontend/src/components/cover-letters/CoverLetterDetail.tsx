/**
 * CoverLetterDetail — full cover letter view with content and metadata.
 *
 * Displays cover letter content, job info, strengths/gaps, and action buttons.
 *
 * @example
 *   <CoverLetterDetail coverLetter={coverLetter} />
 */

"use client";

import { useState } from "react";
import { Mail, Trash2, Sparkles } from "lucide-react";
import type { CoverLetter } from "@/lib/types/resumes";
import { useDeleteCoverLetter } from "@/hooks/useResumes";
import { Button } from "@/components/shared/Button";

interface CoverLetterDetailProps {
  coverLetter: CoverLetter;
}

/**
 * Map API error codes to user-friendly messages.
 */
function getErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    const msg = error.message.toLowerCase();
    if (msg.includes("version_conflict")) {
      return "Cover letter was modified by another process. Please refresh.";
    }
    if (msg.includes("not_found")) {
      return "Cover letter not found.";
    }
  }
  return "Something went wrong. Please try again.";
}

export function CoverLetterDetail({ coverLetter }: CoverLetterDetailProps) {
  const [error, setError] = useState<string | null>(null);
  const deleteMutation = useDeleteCoverLetter();

  const handleDelete = () => {
    if (!window.confirm("Are you sure you want to delete this cover letter?")) return;
    deleteMutation.mutate(coverLetter.id, {
      onError: (err) => setError(getErrorMessage(err)),
    });
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg bg-primary-light">
            <Mail className="h-6 w-6 text-primary" aria-hidden="true" />
          </div>
          <div>
            <h1 className="text-xl font-semibold text-foreground">
              {coverLetter.job_title ?? "Untitled Position"}
            </h1>
            <p className="text-sm text-text-secondary">
              {coverLetter.word_count ?? 0} words
              {coverLetter.model != null && ` · ${coverLetter.model}`}
            </p>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2">
          <Button
            variant="danger"
            size="sm"
            loading={deleteMutation.isPending}
            loadingText="Deleting…"
            onClick={handleDelete}
          >
            <Trash2 className="mr-1 h-4 w-4" aria-hidden="true" />
            Delete
          </Button>
        </div>
      </div>

      {/* Error */}
      {error != null && (
        <div role="alert" className="rounded-md bg-danger-light px-3 py-2 text-sm text-danger-dark">
          {error}
        </div>
      )}

      {/* Content */}
      {coverLetter.content.length > 0 ? (
        <section>
          <h2 className="text-sm font-medium text-text-secondary mb-2">
            Cover Letter
          </h2>
          <div className="rounded-lg border border-border bg-bg-secondary p-4">
            <div className="whitespace-pre-wrap text-sm text-foreground leading-relaxed">
              {coverLetter.content}
            </div>
          </div>
        </section>
      ) : (
        <div className="rounded-lg border border-dashed border-border p-8 text-center">
          <Sparkles className="mx-auto h-8 w-8 text-text-tertiary" aria-hidden="true" />
          <p className="mt-2 text-sm text-text-secondary">
            No content generated yet.
          </p>
        </div>
      )}

      {/* Strengths */}
      {coverLetter.strengths != null && coverLetter.strengths.length > 0 && (
        <section>
          <h2 className="text-sm font-medium text-text-secondary mb-2">
            Strengths
          </h2>
          <ul className="space-y-1">
            {coverLetter.strengths.map((s, i) => (
              <li
                key={i}
                className="text-sm text-text-secondary list-disc list-inside"
              >
                {s}
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* Gaps */}
      {coverLetter.gaps != null && coverLetter.gaps.length > 0 && (
        <section>
          <h2 className="text-sm font-medium text-text-secondary mb-2">
            Gaps to Address
          </h2>
          <ul className="space-y-1">
            {coverLetter.gaps.map((g, i) => (
              <li
                key={i}
                className="text-sm text-text-secondary list-disc list-inside"
              >
                {g}
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* Metadata footer */}
      <div className="flex items-center justify-between border-t border-border pt-4 text-xs text-text-tertiary">
        <span>Version {coverLetter.version}</span>
        <span>Updated {new Date(coverLetter.updated_at).toLocaleDateString()}</span>
      </div>
    </div>
  );
}
