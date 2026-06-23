/**
 * Cover Letter detail page — Server Component wrapper.
 */

import { CoverLetterDetailPageClient } from "./CoverLetterDetailPageClient";

interface CoverLetterDetailPageProps {
  params: Promise<{ id: string }>;
}

export default async function CoverLetterDetailPage({ params }: CoverLetterDetailPageProps) {
  const { id } = await params;
  return <CoverLetterDetailPageClient id={id} />;
}
