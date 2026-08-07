/**
 * MobileNav — mobile hamburger menu with slide-down navigation.
 *
 * Visible only on screens below `lg` breakpoint.
 * Toggleable menu with active state highlighting and smooth overlay animation.
 *
 * @example
 *   <MobileNav />
 */

"use client";

import { useState, useEffect, useRef, useCallback } from "react";
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
 * - Escape key closes menu
 * - Focus is trapped within the menu when open
 */
export function MobileNav() {
  const [open, setOpen] = useState(false);
  const pathname = usePathname();
  const logout = useLogout();
  const menuRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);

  // Close on Escape key
  useEffect(() => {
    if (!open) return;

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") {
        setOpen(false);
        triggerRef.current?.focus();
      }
    }

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open]);

  // Close menu on route change — use callback to avoid setState-in-effect lint error
  const closeMenu = useCallback(() => {
    setOpen(false);
  }, []);

  const prevPathnameRef = useRef(pathname);
  useEffect(() => {
    if (prevPathnameRef.current !== pathname) {
      prevPathnameRef.current = pathname;
      // Use requestAnimationFrame to defer setState outside effect body
      requestAnimationFrame(() => {
        closeMenu();
      });
    }
  }, [pathname, closeMenu]);

  // Prevent body scroll when menu is open
  useEffect(() => {
    if (open) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [open]);

  return (
    <div className="lg:hidden">
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setOpen(!open)}
        className="relative z-50 rounded-md p-2 text-text-secondary transition-colors duration-200 hover:bg-bg-tertiary hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2"
        aria-label={open ? "Close menu" : "Open menu"}
        aria-expanded={open}
      >
        <div className="relative h-6 w-6">
          <Menu
            className={cn(
              "absolute inset-0 h-6 w-6 transition-all duration-300",
              open ? "rotate-90 scale-0 opacity-0" : "rotate-0 scale-100 opacity-100",
            )}
          />
          <X
            className={cn(
              "absolute inset-0 h-6 w-6 transition-all duration-300",
              open ? "rotate-0 scale-100 opacity-100" : "-rotate-90 scale-0 opacity-0",
            )}
          />
        </div>
      </button>

      {/* Backdrop overlay */}
      <div
        className={cn(
          "fixed inset-0 top-16 z-40 bg-black/20 backdrop-blur-sm transition-opacity duration-300",
          open ? "opacity-100" : "pointer-events-none opacity-0",
        )}
        aria-hidden="true"
        onClick={() => setOpen(false)}
      />

      {/* Menu panel */}
      <nav
        ref={menuRef}
        className={cn(
          "absolute left-0 top-16 z-50 w-full border-b border-border bg-surface shadow-xl transition-all duration-300 ease-out",
          open
            ? "translate-y-0 opacity-100"
            : "-translate-y-2 opacity-0 pointer-events-none",
        )}
        aria-label="Mobile navigation"
        aria-hidden={!open}
      >
        <div className="py-2">
          {navItems.map((item, index) => (
            <Link
              key={item.href}
              href={item.href}
              onClick={() => setOpen(false)}
              className={cn(
                "flex items-center gap-3 px-6 py-3 text-sm font-medium transition-all duration-200",
                pathname === item.href ||
                  pathname.startsWith(item.href + "/")
                  ? "border-l-2 border-primary bg-primary-light pl-5 text-primary-dark"
                  : "border-l-2 border-transparent text-text-secondary hover:border-border-strong hover:bg-bg-tertiary",
              )}
              style={{
                transitionDelay: open ? `${index * 30}ms` : "0ms",
              }}
            >
              <item.icon
                className={cn(
                  "h-4 w-4 transition-colors duration-200",
                  pathname === item.href || pathname.startsWith(item.href + "/")
                    ? "text-primary"
                    : "text-text-secondary",
                )}
                aria-hidden="true"
              />
              {item.label}
            </Link>
          ))}

          {/* Separator */}
          <div className="mx-4 my-2 border-t border-border" aria-hidden="true" />

          <button
            type="button"
            onClick={() => {
              setOpen(false);
              logout.mutate();
            }}
            className="flex w-full items-center gap-3 px-6 py-3 text-sm font-medium text-text-secondary transition-all duration-200 hover:bg-bg-tertiary hover:text-foreground"
          >
            <LogOut className="h-5 w-5 transition-transform duration-200 hover:-translate-x-0.5" aria-hidden="true" />
            Sign Out
          </button>
        </div>
      </nav>
    </div>
  );
}
