/**
 * API client for the System Config domain.
 *
 * Handles GET/PATCH/DELETE for runtime configuration overrides.
 * The GET endpoint returns the fully resolved EffectiveConfig merging
 * YAML defaults, env vars, and DB overrides.
 *
 * Does NOT cache — use TanStack Query for caching.
 *
 * @see backend/internal/systemconfig/handler.go
 */

import { apiGetWithRefresh, apiPatchWithRefresh, apiDeleteWithRefresh } from "@/lib/api/client";
import type {
  SystemConfigResponse,
  SetOverrideRequest,
  SetOverrideResponse,
  DeleteOverrideResponse,
} from "@/lib/types/config";

/**
 * Fetch the fully resolved system configuration.
 *
 * Returns the merged config tree with sources tracking which layer
 * (yaml/env/db) produced each value.
 *
 * @returns System config response with effective config and optional version
 * @throws ApiError on non-2xx responses
 *
 * @example
 *   const { config, version } = await fetchSystemConfig();
 *   console.log(config.scoring.mode); // "hybrid"
 */
export async function fetchSystemConfig(): Promise<SystemConfigResponse> {
  const result = await apiGetWithRefresh<SystemConfigResponse>("system/config");
  if (result === undefined) {
    throw new Error("Failed to fetch system config");
  }
  return result;
}

/**
 * Set a configuration override.
 *
 * Creates or updates a runtime configuration override. The key must be
 * in the allowlist and the value must pass key-specific validation.
 *
 * @param key - Dotted-notation config key (e.g., "scoring.auto_threshold")
 * @param value - JSON value to store
 * @returns Override response with message and key
 * @throws ApiError on invalid key/format/value
 *
 * @example
 *   await setOverride("scoring.auto_threshold", 95);
 *   await setOverride("llm.primary.model", "gpt-4o-mini");
 */
export async function setOverride(
  key: string,
  value: unknown,
): Promise<SetOverrideResponse> {
  const result = await apiPatchWithRefresh<SetOverrideResponse>("system/config", {
    key,
    value,
  } as SetOverrideRequest);
  if (result === undefined) {
    throw new Error("Failed to set override");
  }
  return result;
}

/**
 * Delete a configuration override.
 *
 * Removes a runtime configuration override by key. The config reverts
 * to YAML/env defaults after deletion.
 *
 * @param key - Dotted-notation config key to delete
 * @returns Delete response with message and key
 * @throws ApiError on invalid key format
 *
 * @example
 *   await deleteOverride("scoring.auto_threshold");
 */
export async function deleteOverride(
  key: string,
): Promise<DeleteOverrideResponse> {
  const result = await apiDeleteWithRefresh<DeleteOverrideResponse>(
    `system/config/${encodeURIComponent(key)}`,
  );
  if (result === undefined) {
    throw new Error("Failed to delete override");
  }
  return result;
}
