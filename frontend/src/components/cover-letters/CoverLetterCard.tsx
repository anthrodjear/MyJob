/**
 * CoverLetterCard — displays a single cover letter in a list.
 *
 * Shows job title, word count, model used, and content preview.
 * Clicking navigates to the cover letter detail page.
 *
 * @example
 *   <CoverLetterCard coverLetter={coverLetter} />
 */

"use client";

import Link from "next/link";
import { Mail, FileText } from "lucide-react";
import type { CoverLetter } from "@/lib/types/resumes";
import { cn } from "@/lib/utils";

interface CoverLetterCardProps {
  coverLetter: CoverLetter;
  className?: string;
}

export function CoverLetterCard({ coverLetter, className }: CoverLetterCardProps) {
  const preview =
    coverLetter.content.length > 150
      ? `${coverLetter.content.slice(0, 150)}…`
      : coverLetter.content;

  return (
    <Link
      href={`/dashboard/cover-letters/${coverLetter.id}`}
      className={cn(
        "block rounded-lg border border-border bg-bg-secondary p-4 transition-colors hover:border-primary/30",
        className,
      )}
    >
      <div className="flex items-start gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary-light">
          <Mail className="h-5 w-5 text-primary" aria-hidden="true" />
        </div>
        <div className="min-w-0 flex-1">
          <h3 className="truncate text-sm font-medium text-foreground">
            {coverLetter.job_title ?? "Untitled Position"}
          </h3>
          <p className="mt-1 text-xs text-text-secondary line-clamp-3">
            {preview}
          </p>
        </div>
      </div>

      {/* Footer */}
      <div className="mt-3 flex items-center justify-between text-xs text-text-tertiary">
        <div className="flex items-center gap-3">
          {coverLetter.word_count != null && (
            <span>{coverLetter.word_count} words</span>
          )}
          {coverLetter.model != null && (
            <span className="truncate max-w-[120px]">{coverLetter.model}</span>
          )}
        </div>
        <span>v{coverLetter.version}</span>
      </div>
    </Link>
  );
}
