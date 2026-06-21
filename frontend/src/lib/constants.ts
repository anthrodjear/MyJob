/** Application status → display config mapping. Aligned with backend/internal/applications/model.go. */
export const APPLICATION_STATUS = {
  draft: { label: "Draft", color: "bg-bg-tertiary text-text-secondary" },
  queued: { label: "Queued", color: "bg-info-light text-info-dark" },
  applied: { label: "Applied", color: "bg-info-light text-info-dark" },
  assessment: { label: "Assessment", color: "bg-warning-light text-warning-dark" },
  phone_screen: { label: "Phone Screen", color: "bg-primary-light text-primary-dark" },
  technical: { label: "Technical", color: "bg-primary-light text-primary-dark" },
  final: { label: "Final", color: "bg-primary-light text-primary-dark" },
  offer: { label: "Offer", color: "bg-success-light text-success-dark" },
  rejected: { label: "Rejected", color: "bg-danger-light text-danger-dark" },
} as const;

export type ApplicationStatusKey = keyof typeof APPLICATION_STATUS;

/** Email classification → display config mapping. Aligned with backend/internal/emails/model.go. */
export const EMAIL_CLASSIFICATION = {
  interview_invite: { label: "Interview Invite", color: "bg-primary-light text-primary-dark" },
  rejection: { label: "Rejection", color: "bg-danger-light text-danger-dark" },
  offer: { label: "Offer", color: "bg-success-light text-success-dark" },
  follow_up: { label: "Follow Up", color: "bg-warning-light text-warning-dark" },
  spam: { label: "Spam", color: "bg-bg-tertiary text-text-tertiary" },
  phishing: { label: "Phishing", color: "bg-danger-light text-danger-dark" },
  other: { label: "Other", color: "bg-bg-tertiary text-text-secondary" },
} as const;

/** Job source → color config for source badges. */
export const SOURCE_COLORS = {
  indeed: "bg-source-indeed text-white",
  greenhouse: "bg-source-greenhouse text-white",
  lever: "bg-source-lever text-white",
  remoteok: "bg-source-remoteok text-white",
  linkedin: "bg-source-linkedin text-white",
  custom: "bg-source-custom text-white",
} as const;

export type SourceKey = keyof typeof SOURCE_COLORS;

/** Default pagination limits. */
export const DEFAULT_PAGE_SIZE = 20;
export const MAX_PAGE_SIZE = 100;

/** Match score thresholds for tier classification. */
export const MATCH_THRESHOLDS = {
  HIGH: 80,
  MEDIUM: 50,
} as const;

/** Polling intervals (ms). */
export const POLL_INTERVAL = {
  tasks: 5000,
  emails: 30000,
  dashboard: 60000,
} as const;

/** API path prefix. */
export const API_PREFIX = "/api/v1" as const;
