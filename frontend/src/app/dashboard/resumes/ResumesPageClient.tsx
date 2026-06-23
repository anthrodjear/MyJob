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
import { CardSkeleton } from "@/components/shared/LoadingSkeleton";
import { Pagination } from "@/components/shared/Pagination";

const PAGE_SIZE = 20;

interface ResumesPageClientProps {
  initialOffset?: number;
}

export function ResumesPageClient({ initialOffset = 0 }: ResumesPageClientProps) {
  const router = useRouter();
  const [offset, setOffset] = useState(initialOffset);
  const currentPage = Math.floor(offset / PAGE_SIZE) + 1;

  const { data, isLoading, error } = useResumes({
    limit: PAGE_SIZE,
    offset,
  });

  const resumes = data?.resumes ?? [];
  const total = data?.total ?? 0;

  if (error != null) {
    return (
      <div role="alert" className="rounded-md bg-danger-light px-3 py-2 text-sm text-danger-dark">
        Failed to load resumes. Please try again.
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-foreground">Resumes</h1>
        <p className="text-sm text-text-secondary">
          Manage your resumes and generate AI-tailored content.
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
          <ResumeList resumes={resumes} />
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
        </>
      )}
    </div>
  );
}
