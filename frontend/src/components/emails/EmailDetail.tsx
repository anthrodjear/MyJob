/**
 * EmailDetail — full email view with actions.
 *
 * Shows email metadata (from, date, classification), body content,
 * reply draft, and action buttons (mark read/unread, classify).
 *
 * @example
 *   <EmailDetail email={email} onBack={() => setSelected(null)} />
 */

"use client";

import { ArrowLeft, CheckCircle, Undo2 } from "lucide-react";
import { formatDate } from "@/lib/utils";
import { ClassificationBadge } from "./ClassificationBadge";
import { Button } from "@/components/shared/Button";
import { useUpdateEmail, useClassifyEmail } from "@/hooks/useEmails";
import type { Email } from "@/lib/types/emails";

interface EmailDetailProps {
  /** Email record to display. */
  email: Email;
  /** Callback to navigate back. */
  onBack?: () => void;
}

export function EmailDetail({ email, onBack }: EmailDetailProps) {
  const updateEmail = useUpdateEmail();
  const classifyEmail = useClassifyEmail();

  return (
    <div className="space-y-6">
      {onBack && (
        <Button
          variant="ghost"
          size="sm"
          onClick={onBack}
          className="gap-1.5"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          Back
        </Button>
      )}

      <div className="space-y-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold">
              {email.subject ?? "No subject"}
            </h2>
            <div className="mt-1 flex items-center gap-2 text-sm text-muted-foreground">
              <span>From: {email.from_address}</span>
              <span aria-hidden="true">·</span>
              <time dateTime={email.received_at}>
                {formatDate(email.received_at)}
              </time>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {email.classification && (
              <ClassificationBadge classification={email.classification} />
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={() =>
              updateEmail.mutate({
                id: email.id,
                data: { is_read: !email.is_read },
              })
            }
            loading={updateEmail.isPending}
            loadingText={email.is_read ? "Marking unread…" : "Marking read…"}
          >
            {email.is_read ? (
              <>
                <Undo2 className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                Mark unread
              </>
            ) : (
              <>
                <CheckCircle className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                Mark read
              </>
            )}
          </Button>

          {!email.classification && (
            <Button
              variant="secondary"
              size="sm"
              onClick={() => classifyEmail.mutate(email.id)}
              loading={classifyEmail.isPending}
              loadingText="Classifying…"
            >
              Classify
            </Button>
          )}
        </div>

        {updateEmail.isError && (
          <p className="text-xs text-destructive" role="alert">
            Failed to update. Please try again.
          </p>
        )}
        {classifyEmail.isError && (
          <p className="text-xs text-destructive" role="alert">
            Failed to classify. Please try again.
          </p>
        )}
      </div>

      <div className="rounded-lg border bg-card p-6">
        {email.body ? (
          <div className="prose prose-sm dark:prose-invert max-w-none whitespace-pre-wrap">
            {email.body}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground italic">No body content</p>
        )}
      </div>

      {email.reply_draft && (
        <div className="space-y-2">
          <h3 className="text-sm font-medium">Draft Reply</h3>
          <div className="rounded-lg border bg-muted/50 p-4">
            <p className="whitespace-pre-wrap text-sm">{email.reply_draft}</p>
          </div>
        </div>
      )}
    </div>
  );
}
