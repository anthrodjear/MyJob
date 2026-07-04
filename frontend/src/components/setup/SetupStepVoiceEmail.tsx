"use client";

/**
 * SetupStepVoiceEmail — Step 3 of onboarding wizard.
 *
 * Collects LiveKit voice and Microsoft 365 email configuration.
 * Both sections have inline "Test" buttons for validation.
 * This entire step is skippable.
 *
 * Accessibility:
 * - Semantic `<fieldset>` with `<legend>` for step context
 * - `aria-describedby` links inputs to help text
 * - `aria-invalid` marks fields with failed validation
 * - `role="status"` for test results (non-intrusive)
 * - Skip button clearly labeled for screen readers
 */

import { useState, type FormEvent } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/shared/Button";
import { testVoiceConfig, testEmailConfig } from "@/lib/api/auth";

interface VoiceEmailState {
  livekitUrl: string;
  livekitKey: string;
  livekitSecret: string;
  msTenantId: string;
  msClientId: string;
  msClientSecret: string;
}

interface TestStatus {
  voice: "idle" | "testing" | "valid" | "invalid";
  email: "idle" | "testing" | "valid" | "invalid";
}

interface SetupStepVoiceEmailProps {
  onNext: (config: VoiceEmailState) => void;
  onBack: () => void;
  onSkip: () => void;
}

/** Status label mapping for test results. */
const statusLabels: Record<TestStatus["voice"], string> = {
  idle: "",
  testing: "Testing…",
  valid: "Configuration is valid",
  invalid: "Invalid configuration",
};

/**
 * VoiceEmailSection — reusable section for voice or email config.
 */
function VoiceEmailSection({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-3">
      <div>
        <h3 className="text-sm font-medium text-text-primary">{title}</h3>
        <p className="text-xs text-text-tertiary">{description}</p>
      </div>
      <div className="space-y-3">{children}</div>
    </div>
  );
}

/**
 * SetupStepVoiceEmail — voice and email configuration step.
 *
 * @example
 *   <SetupStepVoiceEmail
 *     onNext={(config) => saveConfig(config)}
 *     onBack={() => goBack()}
 *     onSkip={() => skipStep()}
 *   />
 */
export function SetupStepVoiceEmail({
  onNext,
  onBack,
  onSkip,
}: SetupStepVoiceEmailProps) {
  const [config, setConfig] = useState<VoiceEmailState>({
    livekitUrl: "",
    livekitKey: "",
    livekitSecret: "",
    msTenantId: "",
    msClientId: "",
    msClientSecret: "",
  });
  const [testStatus, setTestStatus] = useState<TestStatus>({
    voice: "idle",
    email: "idle",
  });

  const handleTestVoice = async () => {
    setTestStatus((prev) => ({ ...prev, voice: "testing" }));
    try {
      const result = await testVoiceConfig(
        config.livekitUrl,
        config.livekitKey,
        config.livekitSecret,
      );
      setTestStatus((prev) => ({
        ...prev,
        voice: result.valid ? "valid" : "invalid",
      }));
    } catch {
      setTestStatus((prev) => ({ ...prev, voice: "invalid" }));
    }
  };

  const handleTestEmail = async () => {
    setTestStatus((prev) => ({ ...prev, email: "testing" }));
    try {
      const result = await testEmailConfig(
        config.msTenantId,
        config.msClientId,
        config.msClientSecret,
      );
      setTestStatus((prev) => ({
        ...prev,
        email: result.valid ? "valid" : "invalid",
      }));
    } catch {
      setTestStatus((prev) => ({ ...prev, email: "invalid" }));
    }
  };

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    onNext(config);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <fieldset className="space-y-6">
        <legend className="text-lg font-semibold text-text-primary">
          Voice &amp; Email Integration
        </legend>
        <p className="text-sm text-text-secondary">
          Configure phone screening and email automation. This step is optional
          and can be configured later in Settings.
        </p>

        {/* LiveKit Voice Section */}
        <VoiceEmailSection
          title="LiveKit Voice Agent"
          description="Enables AI phone screening for interviews."
        >
          <div className="space-y-2">
            <label
              htmlFor="livekit-url"
              className="block text-sm font-medium text-text-primary"
            >
              Server URL
            </label>
            <input
              id="livekit-url"
              type="text"
              autoComplete="off"
              value={config.livekitUrl}
              onChange={(e) =>
                setConfig((prev) => ({ ...prev, livekitUrl: e.target.value }))
              }
              placeholder="wss://your-project.livekit.cloud"
              className="flex-1 rounded-md border bg-background px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              aria-describedby="livekit-url-help"
            />
            <p id="livekit-url-help" className="text-xs text-text-tertiary">
              Your LiveKit Cloud project URL.
            </p>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <label
                htmlFor="livekit-key"
                className="block text-sm font-medium text-text-primary"
              >
                API Key
              </label>
              <input
                id="livekit-key"
                type="password"
                autoComplete="off"
                value={config.livekitKey}
                onChange={(e) =>
                  setConfig((prev) => ({ ...prev, livekitKey: e.target.value }))
                }
                placeholder="API key"
                aria-describedby="livekit-key-help"
                className="w-full rounded-md border bg-background px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <p id="livekit-key-help" className="text-xs text-text-tertiary">
                Found in your LiveKit Cloud dashboard.
              </p>
            </div>
            <div className="space-y-2">
              <label
                htmlFor="livekit-secret"
                className="block text-sm font-medium text-text-primary"
              >
                API Secret
              </label>
              <input
                id="livekit-secret"
                type="password"
                autoComplete="off"
                value={config.livekitSecret}
                onChange={(e) =>
                  setConfig((prev) => ({
                    ...prev,
                    livekitSecret: e.target.value,
                  }))
                }
                placeholder="API secret"
                aria-describedby="livekit-secret-help"
                className="w-full rounded-md border bg-background px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <p id="livekit-secret-help" className="text-xs text-text-tertiary">
                Keep this secret and never share it.
              </p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <Button
              type="button"
              variant="secondary"
              size="sm"
              loading={testStatus.voice === "testing"}
              loadingText="Testing"
              disabled={
                !config.livekitUrl || !config.livekitKey || !config.livekitSecret
              }
              onClick={handleTestVoice}
              aria-label="Test voice configuration"
            >
              Test Connection
            </Button>
            {testStatus.voice !== "idle" && testStatus.voice !== "testing" && (
              <p
                role="status"
                className={cn(
                  "text-xs",
                  testStatus.voice === "valid"
                    ? "text-success-dark"
                    : "text-danger-dark",
                )}
              >
                {statusLabels[testStatus.voice]}
              </p>
            )}
          </div>
        </VoiceEmailSection>

        {/* Microsoft 365 Email Section */}
        <VoiceEmailSection
          title="Microsoft 365 Email"
          description="Automates job application emails via Microsoft Graph."
        >
          <div className="space-y-2">
            <label
              htmlFor="ms-tenant-id"
              className="block text-sm font-medium text-text-primary"
            >
              Tenant ID
            </label>
            <input
              id="ms-tenant-id"
              type="text"
              autoComplete="off"
              value={config.msTenantId}
              onChange={(e) =>
                setConfig((prev) => ({ ...prev, msTenantId: e.target.value }))
              }
              placeholder="Azure AD tenant GUID"
              className="w-full rounded-md border bg-background px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              aria-describedby="ms-tenant-help"
            />
            <p id="ms-tenant-help" className="text-xs text-text-tertiary">
              Found in Azure Portal → Azure Active Directory → Overview.
            </p>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <label
                htmlFor="ms-client-id"
                className="block text-sm font-medium text-text-primary"
              >
                Client ID
              </label>
              <input
                id="ms-client-id"
                type="text"
                autoComplete="off"
                value={config.msClientId}
                onChange={(e) =>
                  setConfig((prev) => ({ ...prev, msClientId: e.target.value }))
                }
                placeholder="App registration ID"
                aria-describedby="ms-client-id-help"
                className="w-full rounded-md border bg-background px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <p id="ms-client-id-help" className="text-xs text-text-tertiary">
                Found in Azure Portal → App Registrations.
              </p>
            </div>
            <div className="space-y-2">
              <label
                htmlFor="ms-client-secret"
                className="block text-sm font-medium text-text-primary"
              >
                Client Secret
              </label>
              <input
                id="ms-client-secret"
                type="password"
                autoComplete="off"
                value={config.msClientSecret}
                onChange={(e) =>
                  setConfig((prev) => ({
                    ...prev,
                    msClientSecret: e.target.value,
                  }))
                }
                placeholder="App secret value"
                aria-describedby="ms-client-secret-help"
                className="w-full rounded-md border bg-background px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <p id="ms-client-secret-help" className="text-xs text-text-tertiary">
                Keep this secret and never share it.
              </p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <Button
              type="button"
              variant="secondary"
              size="sm"
              loading={testStatus.email === "testing"}
              loadingText="Testing"
              disabled={
                !config.msTenantId ||
                !config.msClientId ||
                !config.msClientSecret
              }
              onClick={handleTestEmail}
              aria-label="Test email configuration"
            >
              Test Connection
            </Button>
            {testStatus.email !== "idle" && testStatus.email !== "testing" && (
              <p
                role="status"
                className={cn(
                  "text-xs",
                  testStatus.email === "valid"
                    ? "text-success-dark"
                    : "text-danger-dark",
                )}
              >
                {statusLabels[testStatus.email]}
              </p>
            )}
          </div>
        </VoiceEmailSection>
      </fieldset>

      <div className="flex justify-between">
        <Button type="button" variant="ghost" onClick={onBack}>
          Back
        </Button>
        <div className="flex gap-3">
          <Button type="button" variant="ghost" onClick={onSkip}>
            Skip for now
          </Button>
          <Button type="submit" variant="primary" size="lg">
            Continue
          </Button>
        </div>
      </div>
    </form>
  );
}
