/**
 * CoverLettersPageClient — cover letters list with pagination.
 *
 * Client Component (uses hooks for data fetching).
 */

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useCoverLetters } from "@/hooks/useResumes";
import { CoverLetterList } from "@/components/cover-letters/CoverLetterList";
import { CardSkeleton } from "@/components/shared/LoadingSkeleton";
import { Pagination } from "@/components/shared/Pagination";

const PAGE_SIZE = 20;

interface CoverLettersPageClientProps {
  initialOffset?: number;
}

export function CoverLettersPageClient({ initialOffset = 0 }: CoverLettersPageClientProps) {
  const router = useRouter();
  const [offset, setOffset] = useState(initialOffset);
  const currentPage = Math.floor(offset / PAGE_SIZE) + 1;

  const { data, isLoading, error } = useCoverLetters({
    limit: PAGE_SIZE,
    offset,
  });

  const coverLetters = data?.cover_letters ?? [];
  const total = data?.total ?? 0;

  if (error != null) {
    return (
      <div role="alert" className="rounded-md bg-danger-light px-3 py-2 text-sm text-danger-dark">
        Failed to load cover letters. Please try again.
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-foreground">Cover Letters</h1>
        <p className="text-sm text-text-secondary">
          AI-generated cover letters tailored to each job application.
        </p>
      </div>

      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <CardSkeleton key={i} />
          ))}
        </div>
      ) : (
        <>
          <CoverLetterList coverLetters={coverLetters} />
          <Pagination
            page={currentPage}
            total={total}
            limit={PAGE_SIZE}
            onPageChange={(page) => {
              const newOffset = (page - 1) * PAGE_SIZE;
              setOffset(newOffset);
              router.push(`/dashboard/cover-letters?offset=${newOffset}`, { scroll: false });
            }}
          />
        </>
      )}
    </div>
  );
}
