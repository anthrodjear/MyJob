/**
 * Card container — surface-level grouping element.
 *
 * Provides Card, CardHeader, CardContent, and CardFooter subcomponents
 * for consistent layout across the dashboard.
 *
 * No `"use client"` — pure presentational, works in Server Components.
 * (Only becomes client when interactive with event handlers.)
 *
 * @example
 *   <Card>
 *     <CardHeader>Job Listing</CardHeader>
 *     <CardContent>Details here</CardContent>
 *     <CardFooter>Actions</CardFooter>
 *   </Card>
 *   <Card padding={false}>Custom padding</Card>
 *   <Card interactive onClick={handleClick} aria-label="View details">Clickable</Card>
 */

import {
  type ComponentPropsWithRef,
  type KeyboardEvent,
  type ReactNode,
} from "react";
import { cn } from "@/lib/utils";

interface CardProps extends Omit<ComponentPropsWithRef<"div">, "role"> {
  /** Card content. */
  children: ReactNode;
  /** Apply default padding (p-6). Default: true. */
  padding?: boolean;
  /** Interactive state — adds hover effect, cursor, focus ring, and button behavior. Default: false. */
  interactive?: boolean;
}

/**
 * Card container — white surface with border, rounded corners, and subtle shadow.
 * Use `padding={false}` when you need full control over internal spacing.
 * Use `interactive` for clickable cards (hover, cursor, focus ring, Enter/Space activation).
 */
export function Card({
  children,
  className,
  padding = true,
  interactive = false,
  onClick,
  onKeyDown,
  ref,
  ...props
}: CardProps) {
  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (interactive && (e.key === "Enter" || e.key === " ")) {
      e.preventDefault();
      onClick?.(e as unknown as React.MouseEvent<HTMLDivElement>);
    }
    onKeyDown?.(e);
  };

  return (
    <div
      ref={ref}
      className={cn(
        "rounded-xl border border-border bg-surface shadow-sm",
        padding && "p-6",
        interactive &&
          "cursor-pointer transition-colors duration-150 hover:border-border-strong hover:bg-surface-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2",
        className,
      )}
      tabIndex={interactive ? 0 : undefined}
      role={interactive ? "button" : undefined}
      onClick={onClick}
      onKeyDown={interactive ? handleKeyDown : onKeyDown}
      {...props}
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
 * Semantic wrapper for consistent spacing and named-subcomponent API.
 */
export function CardContent({ children, className }: CardSectionProps) {
  return <div className={cn(className)}>{children}</div>;
}

/**
 * Card footer — bottom section with top border separator.
 * Typically contains action buttons or secondary information.
 * Mirrors CardHeader's border pattern (border-top vs border-bottom).
 */
export function CardFooter({ children, className }: CardSectionProps) {
  return (
    <div className={cn("mt-4 border-t border-border pt-4", className)}>
      {children}
    </div>
  );
}
