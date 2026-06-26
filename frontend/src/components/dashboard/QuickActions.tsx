/**
 * QuickActions — action buttons for common dashboard tasks.
 *
 * Provides quick navigation to key workflows: Discover Jobs, Review Approvals,
 * Check Email, and Settings.
 * Client Component — uses onClick handlers for navigation.
 */

"use client";

import { Badge } from "@/components/shared/Badge";
import { cn } from "@/lib/utils";
import {
  CheckCircle,
  Mail,
  Settings,
  Search,
} from "lucide-react";

interface QuickAction {
  label: string;
  description: string;
  icon: React.ReactNode;
  href: string;
  variant?: "primary" | "secondary" | "ghost";
}

const actions: QuickAction[] = [
  {
    label: "Discover Jobs",
    description: "Search and score new job postings",
    icon: <Search className="h-5 w-5" />,
    href: "/dashboard/jobs",
    variant: "primary",
  },
  {
    label: "Review Approvals",
    description: "Approve or reject pending applications",
    icon: <CheckCircle className="h-5 w-5" />,
    href: "/dashboard/approvals",
    variant: "secondary",
  },
  {
    label: "Check Email",
    description: "Review classified recruiter emails",
    icon: <Mail className="h-5 w-5" />,
    href: "/dashboard/email",
    variant: "ghost",
  },
  {
    label: "Settings",
    description: "Configure profile, scraper, and matching",
    icon: <Settings className="h-5 w-5" />,
    href: "/dashboard/settings",
    variant: "ghost",
  },
];

/**
 * QuickActions — action button grid.
 *
 * Renders 4 action cards with icons, labels, and descriptions.
 * Uses next/link for client-side navigation.
 */
export function QuickActions() {
  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {actions.map((action) => (
        <a
          key={action.label}
          href={action.href}
          className={cn(
            "flex flex-col items-start p-4 rounded-xl border border-border bg-surface hover:bg-surface-hover transition-colors",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2",
          )}
        >
          <div className="flex items-center justify-between w-full mb-2">
            <span className="text-xl" aria-hidden="true">
              {action.icon}
            </span>
            <Badge variant="default" className="text-xs">
              {action.variant === "primary" && "Primary"}
              {action.variant === "secondary" && "Secondary"}
              {action.variant === "ghost" && "Ghost"}
            </Badge>
          </div>
          <h3 className="font-semibold text-text-primary mb-1">
            {action.label}
          </h3>
          <p className="text-sm text-text-secondary">
            {action.description}
          </p>
        </a>
      ))}
    </div>
  );
}