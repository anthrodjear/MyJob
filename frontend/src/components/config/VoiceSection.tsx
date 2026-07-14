/**
 * VoiceSection — voice provider and LiveKit configuration editor.
 *
 * Covers voice provider, model, and LiveKit URL.
 * Uses controlled form state with local React state. On submit, calls
 * executeOverrides to batch all changes with proper error handling.
 *
 * Does NOT:
 * - Handle Scoring/LLM/Email settings (separate sections)
 * - Manage authentication state
 * - Expose LiveKit API key/secret (set via .env only, never in config overrides)
 *
 * @see lib/types/config.ts — VoiceSection
 * @see hooks/useSystemConfig.ts — useSetOverride, executeOverrides
 */

"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { useSetOverride, executeOverrides } from "@/hooks/useSystemConfig";
import type { VoiceSection as VoiceSectionType } from "@/lib/types/config";
import { Button } from "@/components/shared/Button";

/** Props for VoiceSection. */
interface VoiceSectionProps {
  /** Current voice config to populate the form. */
  voice: VoiceSectionType;
  /** Called after a successful save. */
  onSaved?: () => void;
}

/** Shared input class with consistent styling and surface background. */
const INPUT_CLASS =
  "mt-1 block w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text-primary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary";

/**
 * Form for editing voice configuration.
 *
 * Renders fields for provider and LiveKit connection (URL only).
 * API key/secret are set via .env and never exposed in config overrides.
 *
 * @example
 *   <VoiceSection voice={config.voice} onSaved={() => console.log("saved")} />
 */
export function VoiceSection({ voice, onSaved }: VoiceSectionProps) {
  const { mutateAsync } = useSetOverride();

  const [provider, setProvider] = useState(voice.provider);
  const [model, setModel] = useState(voice.model);
  const [livekitUrl, setLivekitUrl] = useState(voice.livekit.url);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const clearError = useCallback(() => setError(null), []);

  // Sync state when props change (skip initial mount)
  const isInitialMount = useRef(true);
  useEffect(() => {
    if (isInitialMount.current) {
      isInitialMount.current = false;
      return;
    }
    setProvider(voice.provider);
    setModel(voice.model);
    setLivekitUrl(voice.livekit.url);
  }, [voice]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setIsSaving(true);
      setError(null);

      const overrides: Array<[string, string]> = [];

      if (provider !== voice.provider) {
        if (!provider.trim()) {
          setError("Provider name cannot be empty.");
          setIsSaving(false);
          return;
        }
        overrides.push(["voice.provider", provider]);
      }
      if (model !== voice.model) {
        overrides.push(["voice.model", model]);
      }
      if (livekitUrl !== voice.livekit.url) {
        const trimmedUrl = livekitUrl.trim();
        if (trimmedUrl && !/^wss?:\/\//.test(trimmedUrl)) {
          setError("LiveKit URL must start with ws:// or wss://");
          setIsSaving(false);
          return;
        }
        overrides.push(["voice.livekit.url", livekitUrl]);
      }

      try {
        const result = await executeOverrides(overrides, mutateAsync, onSaved);
        if (result.failed > 0) {
          setError(
            result.failed === result.total
              ? "Failed to save voice settings. Please try again."
              : `Partially saved: ${result.succeeded} of ${result.total} settings saved. ${result.failed} failed.`,
          );
        }
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : "Failed to save voice settings. Please try again.",
        );
      } finally {
        setIsSaving(false);
      }
    },
    [provider, model, livekitUrl, voice, mutateAsync, onSaved],
  );

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {error && (
        <div className="rounded-md bg-danger-light p-3 text-sm text-danger-dark" role="alert">
          {error}
        </div>
      )}

      {/* Provider Settings */}
      <fieldset className="space-y-4">
        <legend className="text-sm font-medium text-text-primary">Voice Provider</legend>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label htmlFor="voice-provider" className="block text-sm font-medium text-text-primary">
              Provider
            </label>
            <input
              id="voice-provider"
              type="text"
              value={provider}
              onChange={(e) => {
                setProvider(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label htmlFor="voice-model" className="block text-sm font-medium text-text-primary">
              Model
            </label>
            <input
              id="voice-model"
              type="text"
              value={model}
              onChange={(e) => {
                setModel(e.target.value);
                clearError();
              }}
              className={INPUT_CLASS}
            />
          </div>
        </div>
      </fieldset>

      {/* LiveKit Connection */}
      <fieldset className="space-y-4">
        <legend className="text-sm font-medium text-text-primary">LiveKit Connection</legend>
        <div>
          <label htmlFor="livekit-url" className="block text-sm font-medium text-text-primary">
            LiveKit URL
          </label>
          <input
            id="livekit-url"
            type="text"
            value={livekitUrl}
            onChange={(e) => {
              setLivekitUrl(e.target.value);
              clearError();
            }}
            className={INPUT_CLASS}
          />
          <p className="mt-1 text-xs text-text-secondary">
            WebSocket URL for LiveKit server (e.g., ws://localhost:7880)
          </p>
        </div>
        <p className="text-xs text-text-secondary">
          LiveKit API key and secret are configured via environment variables for security.
        </p>
      </fieldset>

      <div className="flex justify-end">
        <Button type="submit" variant="primary" disabled={isSaving} loading={isSaving}>
          Save Voice Settings
        </Button>
      </div>
    </form>
  );
}