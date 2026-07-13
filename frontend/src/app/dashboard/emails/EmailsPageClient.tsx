/**
 * EmailsPageClient — client-side emails page with classification filter.
 *
 * URL-synced filter state via useSearchParams + router.push.
 * Single router.push for multi-param changes to avoid stale snapshot bugs.
 *
 * @example
 *   /dashboard/emails?classification=interview_invite&limit=50
 */

"use client";

import { useState, useCallback, useMemo } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Mail } from "lucide-react";
import { useEmails } from "@/hooks/useEmails";
import { EmailList } from "@/components/emails/EmailList";
import { EmailDetail } from "@/components/emails/EmailDetail";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { Email, EmailClassification } from "@/lib/types/emails";

const CLASSIFICATION_OPTIONS: { value: EmailClassification; label: string }[] = [
  { value: "interview_invite", label: "Interview Invite" },
  { value: "rejection", label: "Rejection" },
  { value: "offer", label: "Offer" },
  { value: "follow_up", label: "Follow Up" },
  { value: "spam", label: "Spam" },
  { value: "phishing", label: "Phishing" },
  { value: "other", label: "Other" },
];

export function EmailsPageClient() {
  const router = useRouter();
  const searchParams = useSearchParams();

  const classification = searchParams.get("classification") as EmailClassification | null;
  const limit = Number(searchParams.get("limit") || "50");
  const offset = Number(searchParams.get("offset") || "0");

  const [selectedEmail, setSelectedEmail] = useState<Email | null>(null);

  const { data, isLoading, isPlaceholderData } = useEmails({
    classification: classification ?? undefined,
    limit,
    offset,
  });

  const emails = useMemo(() => data?.emails ?? [], [data]);

  const handleSelect = useCallback((email: Email) => {
    setSelectedEmail(email);
  }, []);

  const handleBack = useCallback(() => {
    setSelectedEmail(null);
  }, []);

  if (selectedEmail) {
    return (
      <div className="mx-auto max-w-4xl">
        <EmailDetail email={selectedEmail} onBack={handleBack} />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Emails</h1>
          <p className="text-muted-foreground">
            Emails from job portals and applications.
          </p>
        </div>
        <Mail className="h-8 w-8 text-muted-foreground" aria-hidden="true" />
      </div>

      <div className="flex flex-col gap-4 sm:flex-row">
        <Select
          value={classification ?? "all"}
          onValueChange={(value) => {
            const params = new URLSearchParams(searchParams.toString());
            if (value === "all") {
              params.delete("classification");
            } else {
              params.set("classification", value);
            }
            params.delete("offset");
            router.push(`/dashboard/emails?${params.toString()}`, {
              scroll: false,
            });
          }}
        >
          <SelectTrigger className="w-full sm:w-48" aria-label="Filter by classification">
            <SelectValue placeholder="All classifications" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All classifications</SelectItem>
            {CLASSIFICATION_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div aria-live="polite" aria-atomic="true" aria-busy={isLoading && !isPlaceholderData}>
        {data?.total != null && (
          <p className="text-sm text-muted-foreground mb-4">
            {data.total} {data.total === 1 ? "email" : "emails"}
          </p>
        )}
      </div>

      <EmailList
        emails={emails}
        isLoading={isLoading && !isPlaceholderData}
        onSelect={handleSelect}
      />
    </div>
  );
}
