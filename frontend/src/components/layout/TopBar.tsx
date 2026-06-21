/**
 * TopBar — desktop header bar.
 *
 * Fixed header with notification bell and user avatar.
 * Hidden on mobile (MobileNav handles mobile header).
 *
 * @example
 *   <TopBar />
 */

"use client";

import { cn } from "@/lib/utils";
import { Avatar } from "@/components/shared/Avatar";
import { Bell } from "lucide-react";

interface TopBarProps {
  /** Additional CSS classes. */
  className?: string;
}

/**
 * TopBar — desktop header bar.
 *
 * Accessibility:
 * - `<header>` landmark for screen reader navigation
 * - Notification button with `aria-label="Notifications"`
 * - Avatar with `aria-label` (via Avatar component)
 */
export function TopBar({ className }: TopBarProps) {
  return (
    <header
      className={cn(
        "flex h-16 items-center justify-between border-b border-border bg-surface px-6",
        className,
      )}
    >
      {/* Left spacer — can be used for breadcrumbs or search later */}
      <div className="flex-1" />

      {/* Right side — notifications + user */}
      <div className="flex items-center gap-4">
        <button
          type="button"
          className="relative rounded-md p-2 text-text-secondary hover:bg-bg-tertiary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          aria-label="Notifications"
        >
          <Bell className="h-5 w-5" aria-hidden="true" />
        </button>
        <Avatar name="User" size="sm" />
        {/* TODO: Replace with actual user name from auth context when wired */}
      </div>
    </header>
  );
}
