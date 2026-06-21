/**
 * DashboardLayout — shell wrapper for all dashboard pages.
 *
 * Composes the AppShell (Sidebar, TopBar, MobileNav) with a main content area.
 * Server Component — no client-side interactivity.
 */

import { AppShell } from "@/components/layout/AppShell";

interface DashboardLayoutProps {
  children: React.ReactNode;
}

export default function DashboardLayout({
  children,
}: DashboardLayoutProps) {
  return <AppShell>{children}</AppShell>;
}