/**
 * LLMSection — LLM provider configuration editor.
 *
 * Covers primary, local, and embedding provider settings (provider name and model).
 * Uses controlled form state with local React state. On submit, calls
 * executeOverrides to batch all changes with proper error handling.
 *
 * Does NOT:
 * - Handle Scoring/Voice/Email settings (separate sections)
 * - Manage authentication state
 *
 * @see lib/types/config.ts — LLMSection
 * @see hooks/useSystemConfig.ts — useSetOverride, executeOverrides
 */

"use client";

import { useState, useCallback } from "react";
import { useSetOverride, executeOverrides } from "@/hooks/useSystemConfig";
import type { LLMSection as LLMSectionType } from "@/lib/types/config";
import { Button } from "@/components/shared/Button";

/** Props for LLMSection. */
interface LLMSectionProps {
  /** Current LLM config to populate the form. */
  llm: LLMSectionType;
  /** Called after a successful save. */
  onSaved?: () => void;
}

/** Shared input class with consistent styling and surface background. */
const INPUT_CLASS =
  "mt-1 block w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text-primary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary";

/**
 * Form for editing LLM provider configuration.
 *
 * Renders fields for primary, local, and embedding providers.
 * Each field saves independently via PATCH.
 *
 * @example
 *   <LLMSection llm={config.llm} onSaved={() => console.log("saved")} />
 */
export function LLMSection({ llm, onSaved }: LLMSectionProps) {
  const { mutateAsync } = useSetOverride();

  const [primaryProvider, setPrimaryProvider] = useState(llm.primary.provider);
  const [primaryModel, setPrimaryModel] = useState(llm.primary.model);
  const [localProvider, setLocalProvider] = useState(llm.local.provider);
  const [localModel, setLocalModel] = useState(llm.local.model);
  const [embeddingsProvider, setEmbeddingsProvider] = useState(llm.embeddings.provider);
  const [embeddingsModel, setEmbeddingsModel] = useState(llm.embeddings.model);
  const [isSaving, setIsSaving] = useState(false);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setIsSaving(true);

      const overrides: Array<[string, string]> = [];

      if (primaryProvider !== llm.primary.provider) {
        overrides.push(["llm.primary.provider", primaryProvider]);
      }
      if (primaryModel !== llm.primary.model) {
        overrides.push(["llm.primary.model", primaryModel]);
      }
      if (localProvider !== llm.local.provider) {
        overrides.push(["llm.local.provider", localProvider]);
      }
      if (localModel !== llm.local.model) {
        overrides.push(["llm.local.model", localModel]);
      }
      if (embeddingsProvider !== llm.embeddings.provider) {
        overrides.push(["llm.embeddings.provider", embeddingsProvider]);
      }
      if (embeddingsModel !== llm.embeddings.model) {
        overrides.push(["llm.embeddings.model", embeddingsModel]);
      }

      await executeOverrides(overrides, mutateAsync, onSaved);
      setIsSaving(false);
    },
    [
      primaryProvider,
      primaryModel,
      localProvider,
      localModel,
      embeddingsProvider,
      embeddingsModel,
      llm,
      mutateAsync,
      onSaved,
    ],
  );

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div className="rounded-md bg-danger-light p-3 text-sm text-danger-dark" role="alert">
        Failed to save LLM settings. Please try again.
      </div>

      {/* Primary Provider */}
      <fieldset className="space-y-4">
        <legend className="text-sm font-medium text-text-primary">Primary Provider</legend>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label htmlFor="primary-provider" className="block text-sm font-medium text-text-primary">
              Provider
            </label>
            <input
              id="primary-provider"
              type="text"
              value={primaryProvider}
              onChange={(e) => setPrimaryProvider(e.target.value)}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label htmlFor="primary-model" className="block text-sm font-medium text-text-primary">
              Model
            </label>
            <input
              id="primary-model"
              type="text"
              value={primaryModel}
              onChange={(e) => setPrimaryModel(e.target.value)}
              className={INPUT_CLASS}
            />
          </div>
        </div>
      </fieldset>

      {/* Local Provider */}
      <fieldset className="space-y-4">
        <legend className="text-sm font-medium text-text-primary">Local Provider (Ollama)</legend>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label htmlFor="local-provider" className="block text-sm font-medium text-text-primary">
              Provider
            </label>
            <input
              id="local-provider"
              type="text"
              value={localProvider}
              onChange={(e) => setLocalProvider(e.target.value)}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label htmlFor="local-model" className="block text-sm font-medium text-text-primary">
              Model
            </label>
            <input
              id="local-model"
              type="text"
              value={localModel}
              onChange={(e) => setLocalModel(e.target.value)}
              className={INPUT_CLASS}
            />
          </div>
        </div>
      </fieldset>

      {/* Embeddings Provider */}
      <fieldset className="space-y-4">
        <legend className="text-sm font-medium text-text-primary">Embeddings Provider</legend>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label htmlFor="embeddings-provider" className="block text-sm font-medium text-text-primary">
              Provider
            </label>
            <input
              id="embeddings-provider"
              type="text"
              value={embeddingsProvider}
              onChange={(e) => setEmbeddingsProvider(e.target.value)}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label htmlFor="embeddings-model" className="block text-sm font-medium text-text-primary">
              Model
            </label>
            <input
              id="embeddings-model"
              type="text"
              value={embeddingsModel}
              onChange={(e) => setEmbeddingsModel(e.target.value)}
              className={INPUT_CLASS}
            />
          </div>
        </div>
      </fieldset>

      <div className="flex justify-end">
        <Button type="submit" variant="primary" disabled={isSaving} loading={isSaving}>
          Save LLM Settings
        </Button>
      </div>
    </form>
  );
}
