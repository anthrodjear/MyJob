/**
 * AppShell — dashboard layout wrapper.
 *
 * Composes Sidebar (desktop), TopBar (desktop), MobileNav (mobile),
 * and a scrollable main content area into a responsive dashboard shell.
 *
 * No `"use client"` — pure presentational composition.
 *
 * @example
 *   <AppShell>
 *     <DashboardPage />
 *   </AppShell>
 */

import { type ReactNode } from "react";
import { Sidebar } from "./Sidebar";
import { TopBar } from "./TopBar";
import { MobileNav } from "./MobileNav";

interface AppShellProps {
  /** Page content to render in the main area. */
  children: ReactNode;
}

/**
 * AppShell — responsive dashboard layout.
 *
 * Layout:
 * - Desktop (lg+): Fixed sidebar (w-64) + TopBar + scrollable main
 * - Mobile (<lg): MobileNav header + scrollable main (no sidebar)
 *
 * Accessibility:
 * - Skip-to-content link (WCAG 2.4.1) — first focusable element
 * - Sidebar provides `<aside>` landmark
 * - TopBar provides `<header>` landmark
 * - Main content area uses `<main>` with `id="main-content"` for skip link
 * - Overflow handled at appropriate containers
 */
export function AppShell({ children }: AppShellProps) {
  return (
    <div className="flex h-screen overflow-hidden">
      {/* Skip to main content — visually hidden, visible on focus (WCAG 2.4.1) */}
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:z-[9999] focus:bg-primary focus:text-text-inverse focus:p-2 focus:top-2 focus:left-2 focus:rounded"
      >
        Skip to main content
      </a>

      {/* Desktop sidebar */}
      <Sidebar />

      {/* Content area */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Mobile header — only visible below lg */}
        <div className="flex items-center border-b border-border lg:hidden">
          <MobileNav />
          <span className="ml-3 text-lg font-bold text-primary">MyJob</span>
          {/* TODO: Add notification bell + avatar for mobile when auth is wired */}
        </div>

        {/* Desktop header — hidden on mobile */}
        <TopBar className="hidden lg:flex" />

        {/* Scrollable main content */}
        <main id="main-content" className="flex-1 overflow-y-auto p-6">
          {children}
        </main>
      </div>
    </div>
  );
}
