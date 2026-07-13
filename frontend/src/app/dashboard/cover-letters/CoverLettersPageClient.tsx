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
import { Pagination } from "@/components/shared/Pagination";

const PAGE_SIZE = 20;

interface CoverLettersPageClientProps {
  initialOffset?: number;
}

export function CoverLettersPageClient({ initialOffset = 0 }: CoverLettersPageClientProps) {
  const router = useRouter();
  const [offset, setOffset] = useState(initialOffset);
  const currentPage = Math.floor(offset / PAGE_SIZE) + 1;

  const { data, isLoading, isPlaceholderData } = useCoverLetters({
    limit: PAGE_SIZE,
    offset,
  });

  const coverLetters = data?.cover_letters ?? [];
  const total = data?.total ?? 0;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-foreground">Cover Letters</h1>
        <p className="text-sm text-text-secondary">
          AI-generated cover letters tailored to each job application.
        </p>
      </div>

      <CoverLetterList
        coverLetters={coverLetters}
        isLoading={isLoading && !isPlaceholderData}
      />
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
    </div>
  );
}
