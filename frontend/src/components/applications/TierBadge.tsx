/**
 * TierBadge — approval tier display with semantic colors.
 *
 * Server Component. Shows auto/review/reject tier with appropriate styling.
 * Used in application cards and detail views.
 *
 * @example
 *   <TierBadge tier="auto" />
 */

import { cn } from "@/lib/utils";
import type { ApprovalTier } from "@/lib/types/applications";

/** Tier → badge color mapping. */
const TIER_STYLES: Record<ApprovalTier, string> = {
  auto: "bg-success-light text-success-dark",
  review: "bg-warning-light text-warning-dark",
  reject: "bg-danger-light text-danger-dark",
};

/** Tier → human-readable label. */
const TIER_LABELS: Record<ApprovalTier, string> = {
  auto: "Auto",
  review: "Review",
  reject: "Reject",
};

interface TierBadgeProps {
  /** Approval tier value. */
  tier: ApprovalTier;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * TierBadge — approval tier display.
 *
 * Accessibility:
 * - Uses `aria-label` for screen reader announcement
 */
export function TierBadge({ tier, className }: TierBadgeProps) {
  const colorClass = TIER_STYLES[tier] ?? TIER_STYLES.review;
  const label = TIER_LABELS[tier] ?? tier;

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
        colorClass,
        className,
      )}
      aria-label={`Tier: ${label}`}
    >
      {label}
    </span>
  );
}
