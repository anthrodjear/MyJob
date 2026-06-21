import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * Merges Tailwind CSS classes with clsx.
 * Handles conflicting classes (e.g., "p-2 p-4" → "p-4").
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

/**
 * Formats a date string to a human-readable format in UTC.
 * Returns "—" for null/undefined/invalid dates.
 * Uses UTC to prevent timezone-related day shifts.
 */
export function formatDate(date: string | Date | null | undefined): string {
  if (!date) return "—";
  const d = typeof date === "string" ? new Date(date) : date;
  if (Number.isNaN(d.getTime())) return "—";
  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    timeZone: "UTC",
  }).format(d);
}

/**
 * Formats a match score as a clamped percentage string (0–100).
 * Returns "—" for null/undefined scores.
 */
export function formatScore(score: number | null | undefined): string {
  if (score == null) return "—";
  const safe = Math.min(100, Math.max(0, score));
  return `${Math.round(safe)}%`;
}

/** Score tier levels for business logic. */
export type ScoreLevel = "high" | "medium" | "low";

/**
 * Returns the score tier level for a given score.
 * High: ≥80, Medium: 50–79, Low: <50.
 * Use this in business logic; components map to visual styles.
 */
export function scoreLevel(score: number): ScoreLevel {
  if (score >= 80) return "high";
  if (score >= 50) return "medium";
  return "low";
}

/**
 * Truncates a trimmed string to maxLen characters, appending "..." if truncated.
 */
export function truncate(value: string, maxLen: number): string {
  const text = value.trim();
  if (text.length <= maxLen || maxLen <= 0) return text;
  return `${text.slice(0, maxLen)}...`;
}
