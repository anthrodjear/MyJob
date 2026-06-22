/**
 * Emails page — Server Component wrapper.
 *
 * Delegates to EmailsPageClient for interactive functionality.
 */

import { EmailsPageClient } from "./EmailsPageClient";

export default function EmailsPage() {
  return <EmailsPageClient />;
}
