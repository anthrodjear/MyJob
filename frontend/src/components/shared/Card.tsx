/**
 * Card container — surface-level grouping element.
 *
 * Provides Card, CardHeader, and CardContent subcomponents
 * for consistent layout across the dashboard.
 *
 * No `"use client"` — pure presentational, works in Server Components.
 *
 * @example
 *   <Card>
 *     <CardHeader>Job Listing</CardHeader>
 *     <CardContent>Details here</CardContent>
 *   </Card>
 *   <Card padding={false}>Custom padding</Card>
 */

import { type ReactNode } from "react";
import { cn } from "@/lib/utils";

interface CardProps {
  /** Card content. */
  children: ReactNode;
  /** Additional CSS classes. */
  className?: string;
  /** Apply default padding (p-6). Default: true. */
  padding?: boolean;
}

/**
 * Card container — white surface with border, rounded corners, and subtle shadow.
 * Use `padding={false}` when you need full control over internal spacing.
 */
export function Card({ children, className, padding = true }: CardProps) {
  return (
    <div
      className={cn(
        "rounded-xl border border-border bg-surface shadow-sm",
        padding && "p-6",
        className,
      )}
    >
      {children}
    </div>
  );
}

interface CardSectionProps {
  /** Section content. */
  children: ReactNode;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * Card header — top section with bottom border separator.
 * Typically contains the card title and optional actions.
 */
export function CardHeader({ children, className }: CardSectionProps) {
  return (
    <div className={cn("mb-4 border-b border-border pb-4", className)}>
      {children}
    </div>
  );
}

/**
 * Card content — main body area.
 * Thin wrapper for consistent spacing and class merging.
 */
export function CardContent({ children, className }: CardSectionProps) {
  return <div className={cn(className)}>{children}</div>;
}
