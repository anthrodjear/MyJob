/**
 * EmailList — scrollable list of email cards.
 *
 * Handles loading, empty, and list states.
 * Renders emails in a vertical stack using EmailCard for individual display.
 * Uses SkeletonWrapper to enforce min/max display times and prevent pop-ins.
 *
 * @example
 *   <EmailList emails={emails} isLoading={false} onSelect={handleSelect} />
 */

"use client";

import { Inbox } from "lucide-react";
import { EmailCard } from "./EmailCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { EmailCardSkeleton, SkeletonWrapper } from "@/components/shared/LoadingSkeleton";
import type { Email } from "@/lib/types/emails";

interface EmailListProps {
  /** Array of email records to display. */
  emails: Email[];
  /** Whether the list is currently loading. */
  isLoading?: boolean;
  /** Callback when an email card is clicked. */
  onSelect?: (email: Email) => void;
}

/** Skeleton placeholder matching the list layout. */
function EmailListSkeleton() {
  return (
    <div aria-busy="true" aria-label="Loading emails">
      <span className="sr-only" aria-live="polite">Loading emails…</span>
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
          <EmailCardSkeleton key={i} />
        ))}
      </div>
    </div>
  );
}

export function EmailList({ emails, isLoading = false, onSelect }: EmailListProps) {
  // Use SkeletonWrapper to enforce min/max display times and prevent pop-ins
  return (
    <SkeletonWrapper
      isLoading={isLoading}
      skeleton={<EmailListSkeleton />}
      minDisplayMs={300}
      maxDisplayMs={5000}
      ariaLiveRegion="Emails loaded"
    >
      <div className="space-y-3" role="list" aria-label="Emails">
        {/* Empty state */}
        {emails.length === 0 && !isLoading && (
          <EmptyState
            icon={<Inbox className="h-12 w-12" />}
            title="No emails found"
            description="Emails from job portals will appear here."
          />
        )}

        {/* Emails list */}
        {emails.length > 0 && (
          <>
            {emails.map((email) => (
              <div key={email.id} role="listitem">
                <EmailCard email={email} onClick={() => onSelect?.(email)} />
              </div>
            ))}
          </>
        )}
      </div>
    </SkeletonWrapper>
  );
}
