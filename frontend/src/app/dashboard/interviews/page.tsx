/**
 * Interviews page — Server Component wrapper.
 *
 * Delegates to InterviewsPageClient for interactive functionality.
 *
 * @example
 *   /dashboard/interviews?status=active
 */

import { InterviewsPageClient } from "./InterviewsPageClient";

export default function InterviewsPage() {
  return <InterviewsPageClient />;
}
