/**
 * CoverLetterList — renders a list of cover letters with empty state.
 *
 * Uses CoverLetterCard for each item. Shows EmptyState when no cover letters exist.
 *
 * @example
 *   <CoverLetterList coverLetters={coverLetters} />
 */

import { Mail } from "lucide-react";
import type { CoverLetter } from "@/lib/types/resumes";
import { CoverLetterCard } from "./CoverLetterCard";
import { EmptyState } from "@/components/shared/EmptyState";

interface CoverLetterListProps {
  coverLetters: CoverLetter[];
}

export function CoverLetterList({ coverLetters }: CoverLetterListProps) {
  if (coverLetters.length === 0) {
    return (
      <EmptyState
        icon={<Mail className="h-12 w-12" />}
        title="No cover letters yet"
        description="Cover letters are generated when you apply to jobs. Submit an application to get started."
      />
    );
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {coverLetters.map((cl) => (
        <CoverLetterCard key={cl.id} coverLetter={cl} />
      ))}
    </div>
  );
}
