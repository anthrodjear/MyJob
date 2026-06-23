/**
 * DashboardLayout — shell wrapper for all dashboard pages.
 *
 * Wraps children with AuthGuard (client-side JWT check) and
 * AppShell (Sidebar, TopBar, MobileNav).
 *
 * AuthGuard checks localStorage for JWT on mount and redirects
 * to /login if absent. Server Component renders Client Component
 * AuthGuard — this is the standard Next.js pattern.
 */

import { AppShell } from "@/components/layout/AppShell";
import { AuthGuard } from "@/components/auth/AuthGuard";

interface DashboardLayoutProps {
  children: React.ReactNode;
}

export default function DashboardLayout({
  children,
}: DashboardLayoutProps) {
  return (
    <AuthGuard>
      <AppShell>{children}</AppShell>
    </AuthGuard>
  );
}