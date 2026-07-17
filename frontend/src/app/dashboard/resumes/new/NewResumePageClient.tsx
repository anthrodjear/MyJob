/**
 * NewResumePageClient — form for creating a new resume.
 *
 * Client Component (uses hooks for form state + API calls).
 *
 * @example
 *   <NewResumePageClient />
 */

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { createResume } from "@/lib/api/resumes";
import type { CreateResumeRequest } from "@/lib/types/resumes";

export function NewResumePageClient() {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [form, setForm] = useState<CreateResumeRequest>({
    name: "",
    specialization: "",
    template_path: "templates/resume.tex",
    focus_skills: [],
  });

  const [skillInput, setSkillInput] = useState("");

  function addSkill() {
    const skill = skillInput.trim();
    if (skill && !form.focus_skills.includes(skill)) {
      setForm((prev) => ({
        ...prev,
        focus_skills: [...prev.focus_skills, skill],
      }));
      setSkillInput("");
    }
  }

  function removeSkill(skill: string) {
    setForm((prev) => ({
      ...prev,
      focus_skills: prev.focus_skills.filter((s) => s !== skill),
    }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    if (!form.name.trim()) {
      setError("Name is required");
      return;
    }
    if (!form.specialization.trim()) {
      setError("Specialization is required");
      return;
    }
    if (form.focus_skills.length === 0) {
      setError("Add at least one focus skill");
      return;
    }

    setSubmitting(true);
    try {
      const resume = await createResume(form);
      router.push(`/dashboard/resumes/${resume.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create resume");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-foreground">Create Resume</h1>
        <p className="text-sm text-text-secondary">
          Set up a new resume with your specialization and key skills.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        {error && (
          <div className="rounded-md bg-red-50 p-3 text-sm text-red-700">
            {error}
          </div>
        )}

        <div>
          <label htmlFor="name" className="block text-sm font-medium text-foreground mb-1">
            Resume Name
          </label>
          <input
            id="name"
            type="text"
            value={form.name}
            onChange={(e) => setForm((prev) => ({ ...prev, name: e.target.value }))}
            placeholder="e.g. Senior Go Engineer Resume"
            className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
          />
        </div>

        <div>
          <label htmlFor="specialization" className="block text-sm font-medium text-foreground mb-1">
            Specialization
          </label>
          <input
            id="specialization"
            type="text"
            value={form.specialization}
            onChange={(e) => setForm((prev) => ({ ...prev, specialization: e.target.value }))}
            placeholder="e.g. Backend Engineering"
            className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
          />
        </div>

        <div>
          <label htmlFor="template" className="block text-sm font-medium text-foreground mb-1">
            Template Path
          </label>
          <input
            id="template"
            type="text"
            value={form.template_path}
            onChange={(e) => setForm((prev) => ({ ...prev, template_path: e.target.value }))}
            className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-foreground mb-1">
            Focus Skills
          </label>
          <div className="flex gap-2">
            <input
              type="text"
              value={skillInput}
              onChange={(e) => setSkillInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  addSkill();
                }
              }}
              placeholder="e.g. Go, PostgreSQL, Kubernetes"
              className="flex-1 rounded-md border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            />
            <button
              type="button"
              onClick={addSkill}
              className="rounded-md border border-border bg-background px-3 py-2 text-sm hover:bg-accent"
            >
              Add
            </button>
          </div>
          {form.focus_skills.length > 0 && (
            <div className="mt-2 flex flex-wrap gap-2">
              {form.focus_skills.map((skill) => (
                <span
                  key={skill}
                  className="inline-flex items-center gap-1 rounded-full bg-blue-100 px-3 py-1 text-xs text-blue-800"
                >
                  {skill}
                  <button
                    type="button"
                    onClick={() => removeSkill(skill)}
                    className="ml-1 hover:text-blue-600"
                  >
                    &times;
                  </button>
                </span>
              ))}
            </div>
          )}
        </div>

        <div className="flex gap-3 pt-2">
          <button
            type="submit"
            disabled={submitting}
            className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {submitting ? "Creating..." : "Create Resume"}
          </button>
          <button
            type="button"
            onClick={() => router.push("/dashboard/resumes")}
            className="rounded-md border border-border bg-background px-4 py-2 text-sm font-medium hover:bg-accent"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  );
}
