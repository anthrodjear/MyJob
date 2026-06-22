/**
 * ClassificationBadge — email classification label.
 *
 * Displays the classification category with appropriate color styling.
 * Server Component — no "use client" needed.
 *
 * @example
 *   <ClassificationBadge classification="interview_invite" />
 */

import { cn } from "@/lib/utils";
import type { EmailClassification } from "@/lib/types/emails";

const CLASSIFICATION_STYLES: Record<EmailClassification, string> = {
  interview_invite: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300",
  rejection: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300",
  offer: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300",
  follow_up: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300",
  spam: "bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300",
  phishing: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300",
  other: "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300",
};

const CLASSIFICATION_LABELS: Record<EmailClassification, string> = {
  interview_invite: "Interview Invite",
  rejection: "Rejection",
  offer: "Offer",
  follow_up: "Follow Up",
  spam: "Spam",
  phishing: "Phishing",
  other: "Other",
};

interface ClassificationBadgeProps {
  /** Email classification category from backend. */
  classification: EmailClassification;
  /** Additional CSS classes. */
  className?: string;
}

export function ClassificationBadge({
  classification,
  className,
}: ClassificationBadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
        CLASSIFICATION_STYLES[classification],
        className
      )}
      role="status"
      aria-label={`Classification: ${CLASSIFICATION_LABELS[classification]}`}
    >
      {CLASSIFICATION_LABELS[classification]}
    </span>
  );
}
