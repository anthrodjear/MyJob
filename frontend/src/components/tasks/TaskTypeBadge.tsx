/**
 * TaskTypeBadge — displays task type with icon.
 *
 * Maps task types to human-readable labels.
 *
 * @example
 *   <TaskTypeBadge type="job_discovery" />
 */

import { cn } from "@/lib/utils";
import type { TaskType } from "@/lib/types/tasks";

/** Type → label mapping. Aligned with backend/internal/tasks/model.go. */
const TYPE_LABELS: Record<TaskType, string> = {
  job_discovery: "Job Discovery",
  job_scoring: "Job Scoring",
  application_submit: "Application Submit",
  embedding_generate: "Embedding Generate",
  cover_letter_gen: "Cover Letter Gen",
  resume_generate: "Resume Generate",
  resume_tailor: "Resume Tailor",
  email_check: "Email Check",
  interview_prep: "Interview Prep",
  voice_session: "Voice Session",
  fill_form: "Fill Form",
};

interface TaskTypeBadgeProps {
  type: TaskType;
  className?: string;
}

export function TaskTypeBadge({ type, className }: TaskTypeBadgeProps) {
  const label = TYPE_LABELS[type] ?? type;
  return (
    <span
      className={cn(
        "inline-flex items-center rounded bg-primary-light px-1.5 py-0.5 text-xs font-medium text-primary-dark",
        className,
      )}
    >
      {label}
    </span>
  );
}
