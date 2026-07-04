"use client";

/**
 * SetupStepComplete — Step 5 of onboarding wizard.
 *
 * Confirmation step showing what was configured and next steps.
 * Provides "Go to Dashboard" button to complete onboarding.
 *
 * Accessibility:
 * - Semantic content structure
 * - Clear call-to-action with descriptive text
 * - Success state communicated via visual design
 */

import { Button } from "@/components/shared/Button";

interface SetupStepCompleteProps {
  onComplete: () => void;
}

/** Summary items for configured features. */
const COMPLETED_ITEMS = [
  { label: "Admin account", icon: "👤" },
  { label: "AI scoring engine", icon: "⚡" },
  { label: "Job source integrations", icon: "🔍" },
];

/**
 * SetupStepComplete — final confirmation step.
 *
 * @example
 *   <SetupStepComplete onComplete={() => goToDashboard()} />
 */
export function SetupStepComplete({ onComplete }: SetupStepCompleteProps) {
  return (
    <div className="space-y-6 text-center">
      {/* Success icon */}
      <div className="flex justify-center">
        <div className="flex h-16 w-16 items-center justify-center rounded-full bg-success-light">
          <svg
            className="h-8 w-8 text-success-dark"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
            aria-hidden="true"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M5 13l4 4L19 7"
            />
          </svg>
        </div>
      </div>

      {/* Title */}
      <div>
        <h2 className="text-2xl font-bold text-text-primary">
          You&apos;re all set!
        </h2>
        <p className="mt-2 text-sm text-text-secondary">
          Your AI job search agent is ready to start finding opportunities.
        </p>
      </div>

      {/* What was configured */}
      <div className="mx-auto max-w-sm space-y-3 text-left">
        <h3 className="text-sm font-medium text-text-primary">
          What was configured:
        </h3>
        <ul className="space-y-2" role="list">
          {COMPLETED_ITEMS.map((item) => (
            <li
              key={item.label}
              className="flex items-center gap-3 rounded-md bg-bg-tertiary px-3 py-2"
            >
              <span aria-hidden="true">{item.icon}</span>
              <span className="text-sm text-text-primary">{item.label}</span>
            </li>
          ))}
        </ul>
      </div>

      {/* Next steps */}
      <div className="mx-auto max-w-sm space-y-2 text-left">
        <h3 className="text-sm font-medium text-text-primary">Next steps:</h3>
        <ul className="space-y-1 text-sm text-text-secondary" role="list">
          <li>
            <span className="text-text-tertiary">→</span> Review your scoring
            thresholds in Settings
          </li>
          <li>
            <span className="text-text-tertiary">→</span> Add more job sources
            or custom career pages
          </li>
          <li>
            <span className="text-text-tertiary">→</span> Connect voice and
            email for automated applications
          </li>
        </ul>
      </div>

      {/* CTA */}
      <div className="pt-4">
        <Button
          type="button"
          variant="primary"
          size="lg"
          onClick={onComplete}
          className="w-full max-w-xs"
        >
          Go to Dashboard
        </Button>
      </div>

      {/* Reassurance */}
      <p className="text-xs text-text-tertiary">
        All settings can be changed later from the dashboard.
      </p>
    </div>
  );
}
