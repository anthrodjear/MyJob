/**
 * QueryProvider — TanStack Query client wrapper.
 *
 * Provides a stable QueryClient instance to the component tree.
 * Uses lazy initialization via useState to avoid re-creating the client
 * on every render (per TanStack Query v5 stable-query-client rule).
 *
 * @example
 *   <QueryProvider>
 *     <AppShell>...</AppShell>
 *   </QueryProvider>
 */

"use client";

import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

export function QueryProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  // Stable QueryClient via lazy useState initialization.
  // This creates the client once and reuses it across renders.
  // See: https://tanstack.com/query/v5/docs/eslint/stable-query-client
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            // Cache data for 1 minute before considering stale
            staleTime: 60 * 1000,
            // Retry failed queries once
            retry: 1,
            // Don't refetch on window focus (reduces noise)
            refetchOnWindowFocus: false,
          },
        },
      }),
  );

  return (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}
