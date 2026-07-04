"use client";

/**
 * SetupStepLLMKeys — Step 2 of onboarding wizard.
 *
 * Collects and validates LLM API keys (OpenAI, Anthropic).
 * Each key has an inline "Test" button for validation before proceeding.
 * Keys are optional but recommended — user can skip to next step.
 *
 * Accessibility:
 * - Semantic `<fieldset>` with `<legend>` for step context
 * - `aria-describedby` links inputs to help text
 * - `aria-invalid` marks fields with failed validation
 * - `role="status"` for test results (non-intrusive)
 */

import { useState, type FormEvent } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/shared/Button";
import { testLLMKey } from "@/lib/api/auth";

interface LLMKeyState {
  openai: string;
  anthropic: string;
}

interface TestStatus {
  openai: "idle" | "testing" | "valid" | "invalid";
  anthropic: "idle" | "testing" | "valid" | "invalid";
}

interface SetupStepLLMKeysProps {
  onNext: (keys: LLMKeyState) => void;
  onBack: () => void;
}

/** Status color mapping for test results. */
const statusColors: Record<TestStatus["openai"], string> = {
  idle: "",
  testing: "border-primary",
  valid: "border-success",
  invalid: "border-danger",
};

/** Status label mapping for test results. */
const statusLabels: Record<TestStatus["openai"], string> = {
  idle: "",
  testing: "Testing…",
  valid: "Key is valid",
  invalid: "Invalid key",
};

/** Props for the LLMProviderCard component. */
interface LLMProviderCardProps {
  provider: "openai" | "anthropic";
  label: string;
  placeholder: string;
  value: string;
  testStatus: TestStatus["openai"];
  onValueChange: (value: string) => void;
  onTest: () => void;
}

/**
 * LLM provider configuration card.
 * Renders an input field with test button for a single provider.
 */
function LLMProviderCard({
  provider,
  label,
  placeholder,
  value,
  testStatus,
  onValueChange,
  onTest,
}: LLMProviderCardProps) {

  return (
    <div className="space-y-2">
      <label
        htmlFor={`llm-${provider}`}
        className="block text-sm font-medium text-text-primary"
      >
        {label}
      </label>
      <div className="flex gap-2">
        <input
          id={`llm-${provider}`}
          type="password"
          autoComplete="off"
          value={value}
          onChange={(e) => onValueChange(e.target.value)}
          placeholder={placeholder}
          aria-invalid={testStatus === "invalid" || undefined}
          className={cn(
            "flex-1 rounded-md border bg-background px-3 py-2 text-sm text-text-primary",
            "placeholder:text-text-tertiary",
            "focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary",
            statusColors[testStatus],
          )}
          aria-describedby={`llm-${provider}-help`}
        />
        <Button
          type="button"
          variant="secondary"
          size="md"
          loading={testStatus === "testing"}
          loadingText="Testing"
          disabled={!value}
          onClick={onTest}
          aria-label={`Test ${label} key`}
        >
          Test
        </Button>
      </div>
      {testStatus !== "idle" && testStatus !== "testing" && (
        <p
          role="status"
          className={cn(
            "text-xs",
            testStatus === "valid" ? "text-success-dark" : "text-danger-dark",
          )}
        >
          {statusLabels[testStatus]}
        </p>
      )}
      <p id={`llm-${provider}-help`} className="text-xs text-text-tertiary">
        {provider === "openai"
          ? "Required for GPT-4o job scoring and cover letter generation."
          : "Used as fallback when OpenAI is unavailable."}
      </p>
    </div>
  );
}

/**
 * SetupStepLLMKeys — LLM API key configuration step.
 *
 * @example
 *   <SetupStepLLMKeys
 *     onNext={(keys) => saveKeys(keys)}
 *     onBack={() => goBack()}
 *   />
 */
export function SetupStepLLMKeys({ onNext, onBack }: SetupStepLLMKeysProps) {
  const [keys, setKeys] = useState<LLMKeyState>({ openai: "", anthropic: "" });
  const [testStatus, setTestStatus] = useState<TestStatus>({
    openai: "idle",
    anthropic: "idle",
  });

  const handleTest = async (provider: "openai" | "anthropic") => {
    setTestStatus((prev) => ({ ...prev, [provider]: "testing" }));
    try {
      const result = await testLLMKey(provider, keys[provider]);
      setTestStatus((prev) => ({
        ...prev,
        [provider]: result.valid ? "valid" : "invalid",
      }));
    } catch {
      setTestStatus((prev) => ({ ...prev, [provider]: "invalid" }));
    }
  };

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    onNext(keys);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <fieldset className="space-y-4">
        <legend className="text-lg font-semibold text-text-primary">
          LLM API Keys
        </legend>
        <p className="text-sm text-text-secondary">
          Configure your AI providers. Keys are stored locally and never sent
          externally.
        </p>

        <LLMProviderCard
          provider="openai"
          label="OpenAI API Key"
          placeholder="sk-..."
          value={keys.openai}
          testStatus={testStatus.openai}
          onValueChange={(v) => setKeys((prev) => ({ ...prev, openai: v }))}
          onTest={() => handleTest("openai")}
        />

        <LLMProviderCard
          provider="anthropic"
          label="Anthropic API Key"
          placeholder="sk-ant-..."
          value={keys.anthropic}
          testStatus={testStatus.anthropic}
          onValueChange={(v) => setKeys((prev) => ({ ...prev, anthropic: v }))}
          onTest={() => handleTest("anthropic")}
        />
      </fieldset>

      <div className="flex justify-between">
        <Button type="button" variant="ghost" onClick={onBack}>
          Back
        </Button>
        <Button type="submit" variant="primary" size="lg">
          Continue
        </Button>
      </div>
    </form>
  );
}
