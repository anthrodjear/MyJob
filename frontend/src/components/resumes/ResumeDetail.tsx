/**
 * ResumeDetail — full resume view with content sections.
 *
 * Displays resume metadata, structured content (summary, skills, experience,
 * education, projects), and action buttons (generate, edit, delete).
 *
 * @example
 *   <ResumeDetail resume={resume} />
 */

"use client";

import { useState } from "react";
import {
  FileText,
  Sparkles,
  Pencil,
  Trash2,
  ExternalLink,
} from "lucide-react";
import type { ResumeDetail as ResumeDetailType } from "@/lib/types/resumes";
import { useDeleteResume, useGenerateResumeContent } from "@/hooks/useResumes";
import { Button } from "@/components/shared/Button";
import { cn } from "@/lib/utils";

interface ResumeDetailProps {
  resume: ResumeDetailType;
}

/**
 * Map API error codes to user-friendly messages.
 */
function getErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    const msg = error.message.toLowerCase();
    if (msg.includes("version_conflict")) {
      return "Resume was modified by another process. Please refresh.";
    }
    if (msg.includes("not_found")) {
      return "Resume not found.";
    }
  }
  return "Something went wrong. Please try again.";
}

export function ResumeDetail({ resume }: ResumeDetailProps) {
  const [error, setError] = useState<string | null>(null);
  const deleteMutation = useDeleteResume();
  const generateMutation = useGenerateResumeContent();

  const handleDelete = () => {
    if (!window.confirm("Are you sure you want to delete this resume?")) return;
    deleteMutation.mutate(resume.id, {
      onError: (err) => setError(getErrorMessage(err)),
    });
  };

  const handleGenerate = () => {
    setError(null);
    generateMutation.mutate(
      { id: resume.id },
      { onError: (err) => setError(getErrorMessage(err)) },
    );
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg bg-primary-light">
            <FileText className="h-6 w-6 text-primary" aria-hidden="true" />
          </div>
          <div>
            <h1 className="text-xl font-semibold text-foreground">
              {resume.name}
            </h1>
            <p className="text-sm text-text-secondary">
              {resume.specialization}
            </p>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2">
          <Button
            variant="primary"
            size="sm"
            loading={generateMutation.isPending}
            loadingText="Generating…"
            onClick={handleGenerate}
          >
            <Sparkles className="mr-1 h-4 w-4" aria-hidden="true" />
            Generate
          </Button>
          <Button variant="secondary" size="sm" disabled title="Editing coming soon">
            <Pencil className="mr-1 h-4 w-4" aria-hidden="true" />
            Edit
          </Button>
          <Button
            variant="danger"
            size="sm"
            loading={deleteMutation.isPending}
            loadingText="Deleting…"
            onClick={handleDelete}
          >
            <Trash2 className="mr-1 h-4 w-4" aria-hidden="true" />
            Delete
          </Button>
        </div>
      </div>

      {/* Error */}
      {error != null && (
        <div role="alert" className="rounded-md bg-danger-light px-3 py-2 text-sm text-danger-dark">
          {error}
        </div>
      )}

      {/* Skills */}
      {resume.focus_skills.length > 0 && (
        <section>
          <h2 className="text-sm font-medium text-text-secondary mb-2">
            Focus Skills
          </h2>
          <div className="flex flex-wrap gap-2">
            {resume.focus_skills.map((skill) => (
              <span
                key={skill}
                className="inline-block rounded-full bg-primary-light px-3 py-1 text-sm font-medium text-primary-dark"
              >
                {skill}
              </span>
            ))}
          </div>
        </section>
      )}

      {/* Content sections */}
      {resume.content != null && (
        <div className="space-y-6">
          {/* Summary */}
          {resume.content.summary.length > 0 && (
            <section>
              <h2 className="text-sm font-medium text-text-secondary mb-2">
                Summary
              </h2>
              <p className="text-sm text-foreground leading-relaxed">
                {resume.content.summary}
              </p>
            </section>
          )}

          {/* Skills */}
          {resume.content.skills.length > 0 && (
            <section>
              <h2 className="text-sm font-medium text-text-secondary mb-2">
                Skills
              </h2>
              <div className="flex flex-wrap gap-2">
                {resume.content.skills.map((skill) => (
                  <span
                    key={skill}
                    className="inline-block rounded bg-bg-tertiary px-2 py-1 text-sm text-text-primary"
                  >
                    {skill}
                  </span>
                ))}
              </div>
            </section>
          )}

          {/* Experience */}
          {resume.content.experience.length > 0 && (
            <section>
              <h2 className="text-sm font-medium text-text-secondary mb-3">
                Experience
              </h2>
              <div className="space-y-4">
                {resume.content.experience.map((exp, i) => (
                  <div
                    key={`${exp.company}-${i}`}
                    className="rounded-lg border border-border p-4"
                  >
                    <div className="flex items-start justify-between">
                      <div>
                        <h3 className="text-sm font-medium text-foreground">
                          {exp.title}
                        </h3>
                        <p className="text-sm text-text-secondary">
                          {exp.company}
                          {exp.location.length > 0 && ` · ${exp.location}`}
                        </p>
                      </div>
                      <span className="text-xs text-text-tertiary">
                        {exp.start_date} – {exp.end_date}
                      </span>
                    </div>
                    {exp.description.length > 0 && (
                      <p className="mt-2 text-sm text-text-secondary">
                        {exp.description}
                      </p>
                    )}
                    {exp.highlights.length > 0 && (
                      <ul className="mt-2 space-y-1">
                        {exp.highlights.map((h, j) => (
                          <li
                            key={j}
                            className="text-sm text-text-secondary list-disc list-inside"
                          >
                            {h}
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                ))}
              </div>
            </section>
          )}

          {/* Projects */}
          {resume.content.projects.length > 0 && (
            <section>
              <h2 className="text-sm font-medium text-text-secondary mb-3">
                Projects
              </h2>
              <div className="space-y-3">
                {resume.content.projects.map((project, i) => (
                  <div
                    key={`${project.name}-${i}`}
                    className="rounded-lg border border-border p-4"
                  >
                    <div className="flex items-center justify-between">
                      <h3 className="text-sm font-medium text-foreground">
                        {project.name}
                      </h3>
                      {project.link != null && project.link.length > 0 && (
                        <a
                          href={project.link}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-primary hover:text-primary-hover"
                          aria-label={`Visit ${project.name} (opens in new tab)`}
                        >
                          <ExternalLink className="h-4 w-4" aria-hidden="true" />
                        </a>
                      )}
                    </div>
                    <p className="mt-1 text-sm text-text-secondary">
                      {project.description}
                    </p>
                    {project.technologies.length > 0 && (
                      <div className="mt-2 flex flex-wrap gap-1">
                        {project.technologies.map((tech) => (
                          <span
                            key={tech}
                            className="inline-block rounded bg-bg-tertiary px-1.5 py-0.5 text-xs text-text-secondary"
                          >
                            {tech}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </section>
          )}

          {/* Education */}
          {resume.content.education.length > 0 && (
            <section>
              <h2 className="text-sm font-medium text-text-secondary mb-3">
                Education
              </h2>
              <div className="space-y-3">
                {resume.content.education.map((edu, i) => (
                  <div
                    key={`${edu.institution}-${i}`}
                    className="rounded-lg border border-border p-4"
                  >
                    <h3 className="text-sm font-medium text-foreground">
                      {edu.degree} in {edu.field}
                    </h3>
                    <p className="text-sm text-text-secondary">
                      {edu.institution}
                    </p>
                    <p className="text-xs text-text-tertiary">
                      {edu.start_date} – {edu.end_date}
                      {edu.gpa != null && edu.gpa.length > 0 && ` · GPA: ${edu.gpa}`}
                    </p>
                  </div>
                ))}
              </div>
            </section>
          )}

          {/* Certifications */}
          {resume.content.certifications.length > 0 && (
            <section>
              <h2 className="text-sm font-medium text-text-secondary mb-2">
                Certifications
              </h2>
              <ul className="space-y-1">
                {resume.content.certifications.map((cert, i) => (
                  <li
                    key={i}
                    className="text-sm text-text-secondary list-disc list-inside"
                  >
                    {cert}
                  </li>
                ))}
              </ul>
            </section>
          )}

          {/* Languages */}
          {resume.content.languages.length > 0 && (
            <section>
              <h2 className="text-sm font-medium text-text-secondary mb-2">
                Languages
              </h2>
              <div className="flex flex-wrap gap-3">
                {resume.content.languages.map((lang) => (
                  <span
                    key={lang.language}
                    className="text-sm text-text-secondary"
                  >
                    {lang.language}{" "}
                    <span className="text-text-tertiary">
                      ({lang.proficiency})
                    </span>
                  </span>
                ))}
              </div>
            </section>
          )}

          {/* Links */}
          {resume.content.links.length > 0 && (
            <section>
              <h2 className="text-sm font-medium text-text-secondary mb-2">
                Links
              </h2>
              <div className="flex flex-wrap gap-3">
                {resume.content.links.map((link) => (
                  <a
                    key={link.url}
                    href={link.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 text-sm text-primary hover:text-primary-hover"
                  >
                    <ExternalLink className="h-3 w-3" aria-hidden="true" />
                    {link.label ?? link.type}
                  </a>
                ))}
              </div>
            </section>
          )}
        </div>
      )}

      {/* No content state */}
      {(resume.content == null || !resume.has_content) && (
        <div className="rounded-lg border border-dashed border-border p-8 text-center">
          <Sparkles className="mx-auto h-8 w-8 text-text-tertiary" aria-hidden="true" />
          <p className="mt-2 text-sm text-text-secondary">
            No content generated yet. Click &quot;Generate&quot; to create structured resume content.
          </p>
        </div>
      )}

      {/* Metadata footer */}
      <div className="flex items-center justify-between border-t border-border pt-4 text-xs text-text-tertiary">
        <span>Version {resume.version}</span>
        <span>Updated {new Date(resume.updated_at).toLocaleDateString()}</span>
      </div>
    </div>
  );
}
