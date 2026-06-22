/**
 * Approvals page — main approvals list with filters and pagination.
 *
 * Server Component. Renders ApprovalsPageClient.
 */

import type { Metadata } from "next";
import { ApprovalsPageClient } from "./ApprovalsPageClient";

export const metadata: Metadata = {
  title: "Approvals | MyJob Agent",
  description: "Review and approve job applications.",
};

export default function ApprovalsPage() {
  return <ApprovalsPageClient />;
}
