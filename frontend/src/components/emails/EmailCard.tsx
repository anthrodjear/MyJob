/**
 * EmailCard — clickable email preview card.
 *
 * Shows email metadata (from, subject, date, classification) with read/unread styling.
 * Client Component — uses onClick for interactivity.
 *
 * @example
 *   <EmailCard email={email} onClick={() => setSelected(email)} />
 */

"use client";

import { Mail, MailOpen } from "lucide-react";
import { cn, formatDate } from "@/lib/utils";
import { ClassificationBadge } from "./ClassificationBadge";
import type { Email } from "@/lib/types/emails";

interface EmailCardProps {
  /** Email record to display. */
  email: Email;
  /** Click handler for selection. */
  onClick?: () => void;
}

export function EmailCard({ email, onClick }: EmailCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex w-full items-start gap-4 rounded-lg border p-4 text-left transition-colors",
        "hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        !email.is_read && "border-l-2 border-l-blue-500"
      )}
      aria-label={`Email from ${email.from_address}: ${email.subject ?? "No subject"}`}
    >
      <div className="mt-0.5 shrink-0">
        {email.is_read ? (
          <MailOpen className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
        ) : (
          <Mail className="h-5 w-5 text-blue-500" aria-hidden="true" />
        )}
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between gap-2">
          <p
            className={cn(
              "truncate text-sm",
              email.is_read ? "text-muted-foreground" : "font-semibold"
            )}
          >
            {email.from_address}
          </p>
          <time
            dateTime={email.received_at}
            className="shrink-0 text-xs text-muted-foreground"
          >
            {formatDate(email.received_at)}
          </time>
        </div>

        <p
          className={cn(
            "mt-0.5 truncate text-sm",
            email.is_read ? "text-muted-foreground" : "font-medium"
          )}
        >
          {email.subject ?? "No subject"}
        </p>

        {email.body && (
          <p className="mt-1 truncate text-xs text-muted-foreground">
            {email.body.substring(0, 120)}
          </p>
        )}

        <div className="mt-2 flex items-center gap-2">
          {email.classification && (
            <ClassificationBadge classification={email.classification} />
          )}
          {email.reply_draft && (
            <span
              className="inline-flex items-center rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-800 dark:bg-amber-900/30 dark:text-amber-300"
              aria-label="Has reply draft"
            >
              Draft
            </span>
          )}
        </div>
      </div>
    </button>
  );
}
