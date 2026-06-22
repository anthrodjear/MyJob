/**
 * EmailList — scrollable list of email cards.
 *
 * Handles loading, empty, and list states.
 * Renders emails in a vertical stack using EmailCard for individual display.
 *
 * @example
 *   <EmailList emails={emails} isLoading={false} onSelect={handleSelect} />
 */

"use client";

import { Inbox } from "lucide-react";
import { EmailCard } from "./EmailCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { Skeleton } from "@/components/shared/LoadingSkeleton";
import type { Email } from "@/lib/types/emails";

interface EmailListProps {
  /** Array of email records to display. */
  emails: Email[];
  /** Whether the list is currently loading. */
  isLoading?: boolean;
  /** Callback when an email card is clicked. */
  onSelect?: (email: Email) => void;
}

/** Skeleton placeholder for a single email card during loading. */
function SkeletonCard() {
  return (
    <div className="rounded-lg border p-4">
      <div className="flex items-start gap-4">
        <Skeleton className="h-5 w-5 shrink-0 rounded" />
        <div className="flex-1 space-y-2">
          <Skeleton className="h-4 w-1/3" />
          <Skeleton className="h-4 w-2/3" />
          <Skeleton className="h-3 w-full" />
          <div className="flex gap-2">
            <Skeleton className="h-5 w-20 rounded-full" />
          </div>
        </div>
      </div>
    </div>
  );
}

export function EmailList({ emails, isLoading, onSelect }: EmailListProps) {
  if (isLoading) {
    return (
      <div className="space-y-3" aria-busy="true" aria-label="Loading emails">
        {Array.from({ length: 5 }, (_, i) => (
          <SkeletonCard key={i} />
        ))}
      </div>
    );
  }

  if (emails.length === 0) {
    return (
      <EmptyState
        icon={<Inbox className="h-12 w-12" />}
        title="No emails found"
        description="Emails from job portals will appear here."
      />
    );
  }

  return (
    <div className="space-y-3" role="list" aria-label="Emails">
      {emails.map((email) => (
        <div key={email.id} role="listitem">
          <EmailCard email={email} onClick={() => onSelect?.(email)} />
        </div>
      ))}
    </div>
  );
}
