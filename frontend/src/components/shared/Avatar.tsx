/**
 * Avatar — user profile image or initials fallback.
 *
 * Displays an image when `src` is provided, otherwise renders
 * initials with a deterministic background color derived from the name.
 *
 * No `"use client"` — pure presentational, works in Server Components.
 *
 * @example
 *   <Avatar name="John Doe" src="/avatar.jpg" />
 *   <Avatar name="Jane Smith" size="lg" />
 */

import { cn } from "@/lib/utils";

interface AvatarProps {
  /** User's full name (used for initials and aria-label). */
  name: string;
  /** Image URL. When provided, renders an `<img>` instead of initials. */
  src?: string | null;
  /** Avatar size. Default: "md". */
  size?: "sm" | "md" | "lg";
  /** Additional CSS classes. */
  className?: string;
}

/**
 * Size-to-Tailwind class mapping.
 */
const sizeStyles: Record<"sm" | "md" | "lg", string> = {
  sm: "h-8 w-8 text-xs",
  md: "h-10 w-10 text-sm",
  lg: "h-12 w-12 text-base",
};

/**
 * Extract up to 2 initials from a name.
 * "John Doe" → "JD", "Alice" → "A", "bob jones smith" → "BJ"
 */
function getInitials(name: string): string {
  return name
    .split(" ")
    .map((w) => w[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);
}

/**
 * Deterministic color from a name string.
 * Uses a simple hash to pick from 5 semantic color pairs.
 */
function stringToColor(name: string): string {
  const colors = [
    "bg-primary-light text-primary-dark",
    "bg-success-light text-success-dark",
    "bg-warning-light text-warning-dark",
    "bg-info-light text-info-dark",
    "bg-danger-light text-danger-dark",
  ];
  let hash = 0;
  for (const ch of name) hash = ch.charCodeAt(0) + ((hash << 5) - hash);
  return colors[Math.abs(hash) % colors.length];
}

/**
 * Avatar — user profile image or initials fallback.
 *
 * Accessibility:
 * - `<img>` with `alt={name}` when `src` is provided
 * - `<div>` with `aria-label={name}` for initials fallback
 * - Deterministic color ensures consistent visual identity
 */
export function Avatar({ name, src, size = "md", className }: AvatarProps) {
  if (src != null && src !== "") {
    return (
      <img
        src={src}
        alt={name}
        className={cn(
          "rounded-full object-cover",
          sizeStyles[size],
          className,
        )}
      />
    );
  }

  return (
    <div
      className={cn(
        "flex items-center justify-center rounded-full font-medium",
        sizeStyles[size],
        stringToColor(name),
        className,
      )}
      role="img"
      aria-label={name}
    >
      {getInitials(name)}
    </div>
  );
}
