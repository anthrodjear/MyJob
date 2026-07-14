/**
 * ResumesPageClient — resumes list with pagination.
 *
 * Client Component (uses hooks for data fetching).
 *
 * @example
 *   <ResumesPageClient />
 */

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useResumes } from "@/hooks/useResumes";
import { ResumeList } from "@/components/resumes/ResumeList";
import { Pagination } from "@/components/shared/Pagination";

const PAGE_SIZE = 20;

interface ResumesPageClientProps {
  initialOffset?: number;
}

export function ResumesPageClient({ initialOffset = 0 }: ResumesPageClientProps) {
  const router = useRouter();
  const [offset, setOffset] = useState(initialOffset);
  const currentPage = Math.floor(offset / PAGE_SIZE) + 1;

  const { data, isLoading, isPlaceholderData } = useResumes({
    limit: PAGE_SIZE,
    offset,
  });

  const resumes = data?.resumes ?? [];
  const total = data?.total ?? 0;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-foreground">Resumes</h1>
        <p className="text-sm text-text-secondary">
          Manage your resumes and generate AI-tailored content.
        </p>
      </div>

      <ResumeList
        resumes={resumes}
        isLoading={isLoading && !isPlaceholderData}
      />
      <Pagination
        page={currentPage}
        total={total}
        limit={PAGE_SIZE}
        onPageChange={(page) => {
          const newOffset = (page - 1) * PAGE_SIZE;
          setOffset(newOffset);
          router.push(`/dashboard/resumes?offset=${newOffset}`, { scroll: false });
        }}
      />
    </div>
  );
}
