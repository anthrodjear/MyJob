/**
 * Sidebar — desktop navigation panel.
 *
 * Fixed-width sidebar visible on lg+ screens.
 * Shows navigation links with icons, active state highlighting,
 * and the MyJob brand mark.
 *
 * @example
 *   <Sidebar />
 */

"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { useLogout } from "@/hooks/useAuth";
import {
  LayoutDashboard,
  Briefcase,
  FileText,
  CheckCircle,
  Mail,
  Mic,
  Settings,
  LogOut,
  ScrollText,
  Send,
  ClipboardList,
} from "lucide-react";

/** Navigation items shared between Sidebar and MobileNav. */
export const navItems = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/dashboard/jobs", label: "Jobs", icon: Briefcase },
  { href: "/dashboard/applications", label: "Applications", icon: Send },
  { href: "/dashboard/approvals", label: "Approvals", icon: CheckCircle },
  { href: "/dashboard/resumes", label: "Resumes", icon: ScrollText },
  { href: "/dashboard/cover-letters", label: "Cover Letters", icon: FileText },
  { href: "/dashboard/emails", label: "Emails", icon: Mail },
  { href: "/dashboard/interviews", label: "Interviews", icon: Mic },
  { href: "/dashboard/tasks", label: "Tasks", icon: ClipboardList },
  { href: "/dashboard/settings", label: "Settings", icon: Settings },
] as const;

/**
 * Logout button — clears token and redirects to /login.
 * Uses useLogout hook for consistent logout behavior.
 */
function LogoutButton() {
  const logout = useLogout();

  return (
    <button
      type="button"
      onClick={() => logout.mutate()}
      className="group flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-text-secondary transition-all duration-200 hover:bg-bg-tertiary hover:text-foreground"
    >
      <LogOut className="h-5 w-5 transition-transform duration-200 group-hover:-translate-x-0.5" aria-hidden="true" />
      Sign Out
    </button>
  );
}

interface SidebarProps {
  /** Additional CSS classes. */
  className?: string;
}

/**
 * Sidebar — desktop navigation panel.
 *
 * Accessibility:
 * - `<aside>` landmark for screen reader navigation
 * - `<nav>` wrapper with implicit navigation semantics
 * - Active link highlighted with left border accent + background tint
 * - Icons are decorative (`aria-hidden="true"`)
 * - `usePathname()` for automatic active state
 */
export function Sidebar({ className }: SidebarProps) {
  const pathname = usePathname();

  return (
    <aside
      className={cn(
        "hidden w-64 flex-col border-r border-border bg-bg-secondary lg:flex",
        className,
      )}
    >
      {/* Brand */}
      <div className="relative flex h-16 items-center px-6">
        {/* Subtle gradient accent behind brand */}
        <div
          className="absolute inset-0 bg-gradient-to-r from-primary/5 via-primary/10 to-transparent"
          aria-hidden="true"
        />
        <span className="relative text-xl font-bold tracking-tight text-primary">
          My
          <span className="text-primary-dark">Job</span>
        </span>
      </div>

      {/* Navigation */}
      <nav className="flex-1 space-y-0.5 px-3 py-4" aria-label="Main navigation">
        {navItems.map((item) => {
          const isActive =
            pathname === item.href ||
            pathname.startsWith(item.href + "/");
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "group relative flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 focus-visible:rounded-lg",
                isActive
                  ? "bg-primary-light text-primary-dark"
                  : "text-text-secondary hover:bg-bg-tertiary hover:text-foreground",
              )}
            >
              {/* Active indicator — left border accent */}
              {isActive && (
                <span
                  className="absolute left-0 top-1/2 h-5 w-0.5 -translate-y-1/2 rounded-full bg-primary"
                  aria-hidden="true"
                />
              )}

              {/* Icon with smooth hover transition */}
              <item.icon
                className={cn(
                  "h-5 w-5 transition-all duration-200",
                  isActive
                    ? "text-primary"
                    : "text-text-secondary group-hover:text-foreground group-hover:scale-110",
                )}
                aria-hidden="true"
              />

              {/* Label */}
              <span className="truncate">{item.label}</span>
            </Link>
          );
        })}
      </nav>

      {/* Logout / User section */}
      <div className="mt-auto">
        {/* Separator */}
        <div className="mx-4 border-t border-border" aria-hidden="true" />
        <div className="px-3 py-4">
          <LogoutButton />
        </div>
      </div>
    </aside>
  );
}
