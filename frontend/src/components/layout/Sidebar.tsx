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
      onClick={logout}
      className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-text-secondary transition-colors hover:bg-bg-tertiary hover:text-foreground"
    >
      <LogOut className="h-5 w-5" aria-hidden="true" />
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
 * - Active link highlighted with `bg-primary-light text-primary-dark`
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
      <div className="flex h-16 items-center px-6">
        <span className="text-xl font-bold text-primary">MyJob</span>
      </div>

      {/* Navigation */}
      <nav className="flex-1 space-y-1 px-3 py-4" aria-label="Main navigation">
        {navItems.map((item) => {
          const isActive =
            pathname === item.href ||
            pathname.startsWith(item.href + "/");
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-primary-light text-primary-dark"
                  : "text-text-secondary hover:bg-bg-tertiary hover:text-foreground",
              )}
            >
              <item.icon className="h-5 w-5" aria-hidden="true" />
              {item.label}
            </Link>
          );
        })}
      </nav>

      {/* Logout */}
      <div className="border-t border-border px-3 py-4">
        <LogoutButton />
      </div>
    </aside>
  );
}
