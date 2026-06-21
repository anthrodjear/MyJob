/**
 * UpcomingTasks — pending tasks list for the dashboard.
 *
 * Shows up to 5 pending/running tasks with type, status, and progress.
 * Pure Server Component — receives data as props.
 */

import { type TaskResponse, type TaskStatus, type TaskType } from "@/lib/types/tasks";
import { Card } from "@/components/shared/Card";
import { Badge } from "@/components/shared/Badge";
import { ProgressBar } from "@/components/shared/ProgressBar";
import { formatDate } from "@/lib/utils";
import { cn } from "@/lib/utils";
import { Zap, Cog, FileText, Mail, Mic2, LayoutTemplate, Search, Send, Target } from "lucide-react";

interface UpcomingTasksProps {
  /** Task entries from GET /tasks?status=pending&limit=5 */
  tasks: TaskResponse[];
}

/** Human-readable labels for task types. */
const taskLabels: Record<TaskType, string> = {
  job_discovery: "Job Discovery",
  job_scoring: "Job Scoring",
  application_submit: "Application Submit",
  embedding_generate: "Embedding Generation",
  cover_letter_gen: "Cover Letter Generation",
  resume_generate: "Resume Generation",
  resume_tailor: "Resume Tailoring",
  email_check: "Email Check",
  interview_prep: "Interview Prep",
  voice_session: "Voice Interview",
  fill_form: "Form Filling",
};

/** Icon mapping for task types. */
const taskIcons: Record<TaskType, React.ReactNode> = {
  job_discovery: <Search className="h-5 w-5" aria-hidden="true" />,
  job_scoring: <Target className="h-5 w-5" aria-hidden="true" />,
  application_submit: <Send className="h-5 w-5" aria-hidden="true" />,
  embedding_generate: <Zap className="h-5 w-5" aria-hidden="true" />,
  cover_letter_gen: <FileText className="h-5 w-5" aria-hidden="true" />,
  resume_generate: <FileText className="h-5 w-5" aria-hidden="true" />,
  resume_tailor: <LayoutTemplate className="h-5 w-5" aria-hidden="true" />,
  email_check: <Mail className="h-5 w-5" aria-hidden="true" />,
  interview_prep: <Cog className="h-5 w-5" aria-hidden="true" />,
  voice_session: <Mic2 className="h-5 w-5" aria-hidden="true" />,
  fill_form: <LayoutTemplate className="h-5 w-5" aria-hidden="true" />,
};

/** Status variant mapping for badges. */
const statusVariants: Record<TaskStatus, "default" | "success" | "warning" | "danger" | "info"> = {
  pending: "default",
  running: "info",
  completed: "success",
  failed: "danger",
  cancelled: "default",
};

/** Progress percentage based on task status. */
function getTaskProgress(task: TaskResponse): number {
  switch (task.status) {
    case "completed":
      return 100;
    case "running":
      return 50; // indeterminate, show 50% as "in progress"
    case "failed":
      return 0;
    case "cancelled":
      return 0;
    case "pending":
    default:
      return 0;
  }
}

/**
 * UpcomingTasks — pending/running tasks list.
 *
 * Shows task type, status badge, progress bar, and scheduled time.
 * Empty state when no tasks.
 */
export function UpcomingTasks({ tasks }: UpcomingTasksProps) {
  if (tasks.length === 0) {
    return (
      <Card>
        <div className="flex flex-col items-center justify-center py-8 text-center">
          <Cog className="h-12 w-12 text-text-tertiary mb-2" aria-hidden="true" />
          <p className="text-text-secondary">No upcoming tasks</p>
          <p className="text-sm text-text-tertiary mt-1">
            Tasks will appear here when jobs are discovered or applications are submitted
          </p>
        </div>
      </Card>
    );
  }

  return (
    <Card>
      <div className="mb-4">
        <h3 className="text-lg font-semibold text-text-primary">Upcoming Tasks</h3>
        <p className="text-sm text-text-secondary">
          {tasks.length} task{tasks.length !== 1 ? "s" : ""} pending or running
        </p>
      </div>

      <div className="space-y-3">
        {tasks.map((task) => {
          const progress = getTaskProgress(task);
          const label = taskLabels[task.type] ?? task.type;
          const variant = statusVariants[task.status] ?? "default";
          const icon = taskIcons[task.type] ?? <Cog className="h-5 w-5" aria-hidden="true" />;

          return (
            <div
              key={task.id}
              className="flex items-center gap-3 p-3 rounded-lg bg-bg-secondary"
            >
              <span className="flex-shrink-0 text-text-tertiary" aria-hidden="true">
                {icon}
              </span>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="font-medium text-text-primary truncate">
                    {label}
                  </span>
                  <Badge variant={variant} className="text-xs">
                    {task.status}
                  </Badge>
                  <span className="text-xs text-text-tertiary ml-auto whitespace-nowrap font-mono tabular-nums">
                    Scheduled: {formatDate(task.scheduled_at)}
                  </span>
                </div>

                <ProgressBar
                  value={progress}
                  color={variant === "success" ? "success" : variant === "info" ? "primary" : variant === "danger" ? "danger" : "warning"}
                  className="mt-2"
                />
              </div>
            </div>
          );
        })}
      </div>
    </Card>
  );
}