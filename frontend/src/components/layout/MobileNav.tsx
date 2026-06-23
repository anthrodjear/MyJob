/**
 * MobileNav — mobile hamburger menu with slide-down navigation.
 *
 * Visible only on screens below `lg` breakpoint.
 * Toggleable menu with active state highlighting.
 *
 * @example
 *   <MobileNav />
 */

"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { useLogout } from "@/hooks/useAuth";
import { Menu, X, LogOut } from "lucide-react";
import { navItems } from "./Sidebar";

/**
 * MobileNav — mobile hamburger menu.
 *
 * Accessibility:
 * - Toggle button with `aria-label` ("Open menu" / "Close menu")
 * - `aria-expanded` reflects menu state
 * - Menu closes on link click (improves UX)
 * - `usePathname()` for automatic active state
 */
export function MobileNav() {
  const [open, setOpen] = useState(false);
  const pathname = usePathname();
  const logout = useLogout();

  // Close on Escape key
  useEffect(() => {
    if (!open) return;

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open]);

  return (
    <div className="lg:hidden">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="rounded-md p-2 text-text-secondary hover:bg-bg-tertiary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        aria-label={open ? "Close menu" : "Open menu"}
        aria-expanded={open}
      >
        {open ? <X className="h-6 w-6" /> : <Menu className="h-6 w-6" />}
      </button>
      {open && (
        <nav
          className="absolute left-0 top-16 z-[--z-dropdown] w-full border-b border-border bg-surface shadow-lg"
          aria-label="Mobile navigation"
        >
          {navItems.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              onClick={() => setOpen(false)}
              className={cn(
                "block px-6 py-3 text-sm font-medium",
                pathname === item.href ||
                  pathname.startsWith(item.href + "/")
                  ? "bg-primary-light text-primary-dark"
                  : "text-text-secondary hover:bg-bg-tertiary",
              )}
            >
              {item.label}
            </Link>
          ))}
          <button
            type="button"
            onClick={() => {
              setOpen(false);
              logout();
            }}
            className="flex w-full items-center gap-3 border-t border-border px-6 py-3 text-sm font-medium text-text-secondary hover:bg-bg-tertiary"
          >
            <LogOut className="h-5 w-5" aria-hidden="true" />
            Sign Out
          </button>
        </nav>
      )}
    </div>
  );
}
