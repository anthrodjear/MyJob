/**
 * Applications page — main applications list with filters and pagination.
 *
 * Server Component. Fetches initial data, renders ApplicationList.
 * Client-side hooks handle filtering, pagination, and mutations.
 */

import type { Metadata } from "next";
import { ApplicationsPageClient } from "./ApplicationsPageClient";

export const metadata: Metadata = {
  title: "Applications | MyJob Agent",
  description: "Track and manage your job applications.",
};

export default function ApplicationsPage() {
  return <ApplicationsPageClient />;
}
