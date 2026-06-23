/**
 * Resume detail page — Server Component wrapper.
 *
 * Renders client-side ResumeDetailPageClient for interactivity.
 */

import { ResumeDetailPageClient } from "./ResumeDetailPageClient";

interface ResumeDetailPageProps {
  params: Promise<{ id: string }>;
}

export default async function ResumeDetailPage({ params }: ResumeDetailPageProps) {
  const { id } = await params;
  return <ResumeDetailPageClient id={id} />;
}
