/**
 * ResumeCard — displays a single resume in a list.
 *
 * Shows name, specialization, skills, content status, and version.
 * Clicking the card navigates to the resume detail page.
 *
 * @example
 *   <ResumeCard resume={resume} />
 */

"use client";

import Link from "next/link";
import { FileText, CheckCircle, Clock } from "lucide-react";
import type { Resume } from "@/lib/types/resumes";
import { cn } from "@/lib/utils";

interface ResumeCardProps {
  resume: Resume;
  className?: string;
}

export function ResumeCard({ resume, className }: ResumeCardProps) {
  return (
    <Link
      href={`/dashboard/resumes/${resume.id}`}
      className={cn(
        "block rounded-lg border border-border bg-bg-secondary p-4 transition-colors hover:border-primary/30",
        className,
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-3 min-w-0">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary-light">
            <FileText className="h-5 w-5 text-primary" aria-hidden="true" />
          </div>
          <div className="min-w-0">
            <h3 className="truncate text-sm font-medium text-foreground">
              {resume.name}
            </h3>
            <p className="truncate text-xs text-text-secondary">
              {resume.specialization}
            </p>
          </div>
        </div>

        {/* Content status badge */}
        <span
          className={cn(
            "inline-flex shrink-0 items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
            resume.has_content
              ? "bg-success-light text-success-dark"
              : "bg-bg-tertiary text-text-tertiary",
          )}
        >
          {resume.has_content ? (
            <CheckCircle className="h-3 w-3" aria-hidden="true" />
          ) : (
            <Clock className="h-3 w-3" aria-hidden="true" />
          )}
          {resume.has_content ? "Generated" : "Draft"}
        </span>
      </div>

      {/* Skills */}
      {resume.focus_skills.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-1">
          {resume.focus_skills.slice(0, 5).map((skill) => (
            <span
              key={skill}
              className="inline-block rounded bg-bg-tertiary px-1.5 py-0.5 text-xs text-text-secondary"
            >
              {skill}
            </span>
          ))}
          {resume.focus_skills.length > 5 && (
            <span className="inline-block rounded bg-bg-tertiary px-1.5 py-0.5 text-xs text-text-tertiary">
              +{resume.focus_skills.length - 5}
            </span>
          )}
        </div>
      )}

      {/* Footer */}
      <div className="mt-3 flex items-center justify-between text-xs text-text-tertiary">
        <span>v{resume.version}</span>
        <span>
          {new Date(resume.updated_at).toLocaleDateString()}
        </span>
      </div>
    </Link>
  );
}
