/**
 * Auth API functions — login, password management, and onboarding.
 *
 * Authentication uses HTTP-only session cookies (set by the Route Handler
 * at /api/auth/[...proxy]/route.ts). The client does NOT store tokens in
 * localStorage — all auth state lives in the server-set cookie.
 *
 * Backend endpoints:
 * - POST /auth/login → { access_token, refresh_token, expires_at }
 * - POST /auth/refresh → { access_token, refresh_token, expires_at }
 * - POST /auth/change-password → { message }
 * - POST /auth/logout → { message }
 * - GET /auth/setup/status → { setup_required, step }
 * - POST /auth/setup → { message }
 * - POST /auth/setup/test-llm → { valid }
 * - POST /auth/setup/test-voice → { valid }
 * - POST /auth/setup/test-email → { valid }
 * - POST /auth/setup/config → { message }
 * - POST /auth/setup/onboarding-step → { message }
 * - POST /auth/setup/complete-onboarding → { message }
 *
 * The API client (lib/api/client.ts) automatically sends cookies via
 * credentials: 'include'. The *WithRefresh variants handle 401 → refresh → retry.
 *
 * @example
 *   import { login } from "@/lib/api/auth";
 *   await login("my-password");
 *   // Session cookie is set automatically — all subsequent apiFetch calls
 *   // are authenticated via the cookie
 */

import { apiPost, ApiError } from "./client";

/** Response from POST /auth/change-password and POST /auth/logout. */
export interface MessageResponse {
  message: string;
}

/**
 * Authenticate with the backend via the login Route Handler and set the
 * HTTP-only session cookie.
 *
 * Calls POST /api/auth/login (Next.js Route Handler) which:
 *   1. Proxies to Go backend POST /api/v1/auth/login
 *   2. Encrypts tokens into HTTP-only session cookie
 *   3. Returns success to the client
 *
 * No token storage is needed on the client — the session cookie is
 * automatically managed by the browser.
 *
 * @param password - User's password (single-user local app)
 * @throws ApiError on invalid credentials or server error
 *
 * @example
 *   await login("my-password");
 *   // Session cookie is set — all subsequent apiFetch calls are authenticated
 */
export async function login(password: string): Promise<void> {
  const resp = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ password }),
  });

  if (!resp.ok) {
    const body = (await resp.json().catch(() => ({}))) as {
      error?: string | { code?: string; message?: string };
    };
    const errorObj = typeof body?.error === "object" ? body.error : null;
    const code =
      errorObj?.code ??
      (typeof body?.error === "string" ? body.error : "LOGIN_FAILED");
    const message =
      errorObj?.message ??
      (typeof body?.error === "string" ? body.error : "Login failed");
    throw new ApiError(resp.status, code, message);
  }
}

/**
 * Change the user's password.
 *
 * @param currentPassword - Current password for verification
 * @param newPassword - New password (min 8 characters)
 * @returns Confirmation message
 * @throws ApiError on invalid current password or server error
 *
 * @example
 *   await changePassword("old-pass", "new-strong-pass");
 */
export async function changePassword(
  currentPassword: string,
  newPassword: string,
): Promise<MessageResponse> {
  const resp = await apiPost<MessageResponse>("/auth/change-password", {
    current_password: currentPassword,
    new_password: newPassword,
  });
  if (resp == null) {
    throw new ApiError(500, "EMPTY_RESPONSE", "Password change failed: no response from server");
  }
  return resp;
}

/**
 * Logout — revoke all refresh tokens for the current user on the server.
 *
 * Call this before clearing local auth state to ensure the refresh token
 * cannot be reused. The session cookie is cleared by the Route Handler.
 * The access token is stateless and expires on its own.
 *
 * @returns Confirmation message
 * @throws ApiError on server error (client should still clear session)
 *
 * @example
 *   await logout();
 *   // Session cookie is cleared by the Route Handler
 */
export async function logout(): Promise<MessageResponse> {
  const resp = await apiPost<MessageResponse>("/auth/logout", {});
  if (resp == null) {
    throw new ApiError(500, "EMPTY_RESPONSE", "Logout failed: no response from server");
  }
  return resp;
}

// --- Setup / Onboarding proxy ---
//
// Setup and onboarding endpoints are routed through the Next.js auth proxy
// (/api/auth/setup/*) instead of directly to the Go backend.  This keeps
// all traffic server-side so the client never needs to know the backend URL
// — works regardless of how the app is accessed (localhost, network IP,
// nginx proxy).

/**
 * GET helper for /api/auth/* proxy routes (setup/status etc.).
 * Uses standard fetch — no backend URL construction.
 *
 * Includes a 10s timeout to prevent hanging requests.
 */
async function authProxyGet<T>(path: string): Promise<T> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 10_000);
  try {
    const res = await fetch(`/api/auth/${path}`, {
      method: "GET",
      credentials: "include",
      cache: "no-store",
      signal: controller.signal,
    });
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const code = body?.error?.code ?? "UNKNOWN_ERROR";
      const message = body?.error?.message ?? `Request failed with status ${res.status}`;
      throw new ApiError(res.status, code, message);
    }
    return res.json() as Promise<T>;
  } catch (err) {
    if (err instanceof ApiError) throw err;
    if (err instanceof Error && err.name === "AbortError") {
      throw new ApiError(408, "TIMEOUT", "Request timed out. Please try again.");
    }
    throw new ApiError(500, "PROXY_ERROR", "Something went wrong");
  } finally {
    clearTimeout(timeoutId);
  }
}

/**
 * POST helper for /api/auth/* proxy routes (setup, onboarding etc.).
 * Uses standard fetch — no backend URL construction.
 *
 * Includes a 10s timeout to prevent hanging requests.
 */
async function authProxyPost<T>(path: string, data?: unknown): Promise<T> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 10_000);
  try {
    const res = await fetch(`/api/auth/${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      cache: "no-store",
      signal: controller.signal,
      body: data != null ? JSON.stringify(data) : undefined,
    });
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const code = body?.error?.code ?? "UNKNOWN_ERROR";
      const message = body?.error?.message ?? `Request failed with status ${res.status}`;
      throw new ApiError(res.status, code, message);
    }
    return res.json() as Promise<T>;
  } catch (err) {
    if (err instanceof ApiError) throw err;
    if (err instanceof Error && err.name === "AbortError") {
      throw new ApiError(408, "TIMEOUT", "Request timed out. Please try again.");
    }
    throw new ApiError(500, "PROXY_ERROR", "Something went wrong");
  } finally {
    clearTimeout(timeoutId);
  }
}

// --- Setup API ---

/** Response from GET /auth/setup/status. */
export interface SetupStatusResponse {
  setup_required: boolean;
  step?: string;
  onboarding_completed: boolean;
}

/** Response from POST /auth/setup. */
export interface SetupResponse {
  message: string;
}

/**
 * Check if setup is required (no users exist).
 *
 * @returns Setup status with setup_required flag and optional step for resume
 * @throws ApiError on server error
 */
export async function getSetupStatus(): Promise<SetupStatusResponse> {
  return authProxyGet<SetupStatusResponse>("setup/status");
}

/**
 * Complete the first-boot setup by creating the admin user.
 *
 * @param username - Display name (min 3 chars)
 * @param email - Email address
 * @param password - Password (min 8 chars)
 * @returns Confirmation message
 * @throws ApiError on validation error or if setup already complete
 */
export async function completeSetup(
  username: string,
  email: string,
  password: string,
): Promise<SetupResponse> {
  return authProxyPost<SetupResponse>("setup", { username, email, password });
}

// --- Onboarding API ---

/** Response from POST /auth/setup/test-llm, test-voice, test-email. */
export interface TestServiceResponse {
  valid: boolean;
}

/** Response from POST /auth/setup/config, onboarding-step, complete-onboarding. */
export interface OnboardingResponse {
  message: string;
}

/** Payload for POST /auth/setup/config. */
export interface OnboardingConfigPayload {
  openai_key?: string;
  anthropic_key?: string;
  livekit_url?: string;
  livekit_key?: string;
  livekit_secret?: string;
  ms_tenant_id?: string;
  ms_client_id?: string;
  ms_client_secret?: string;
  auto_threshold?: number;
  review_threshold?: number;
  job_sources?: string[];
  custom_job_sites?: string[];
}

/**
 * Test an LLM API key by calling the provider's validation endpoint.
 *
 * @param provider - "openai" or "anthropic"
 * @param apiKey - The API key to validate
 * @returns Whether the key is valid
 * @throws ApiError on server error
 */
export async function testLLMKey(
  provider: "openai" | "anthropic",
  apiKey: string,
): Promise<TestServiceResponse> {
  return authProxyPost<TestServiceResponse>("setup/test-llm", {
    provider,
    api_key: apiKey,
  });
}

/**
 * Test LiveKit voice configuration by listing rooms.
 *
 * @param url - LiveKit server URL
 * @param apiKey - LiveKit API key
 * @param apiSecret - LiveKit API secret
 * @returns Whether the configuration is valid
 * @throws ApiError on server error
 */
export async function testVoiceConfig(
  url: string,
  apiKey: string,
  apiSecret: string,
): Promise<TestServiceResponse> {
  return authProxyPost<TestServiceResponse>("setup/test-voice", {
    url,
    api_key: apiKey,
    api_secret: apiSecret,
  });
}

/**
 * Test Microsoft 365 email configuration via OAuth token flow.
 *
 * @param tenantId - Azure AD tenant ID
 * @param clientId - App registration client ID
 * @param clientSecret - App registration client secret
 * @returns Whether the configuration is valid
 * @throws ApiError on server error
 */
export async function testEmailConfig(
  tenantId: string,
  clientId: string,
  clientSecret: string,
): Promise<TestServiceResponse> {
  return authProxyPost<TestServiceResponse>("setup/test-email", {
    tenant_id: tenantId,
    client_id: clientId,
    client_secret: clientSecret,
  });
}

/**
 * Save onboarding configuration (LLM keys, voice, email settings).
 *
 * @param config - Configuration payload with optional fields
 * @returns Confirmation message
 * @throws ApiError on server error
 */
export async function saveOnboardingConfig(
  config: OnboardingConfigPayload,
): Promise<OnboardingResponse> {
  return authProxyPost<OnboardingResponse>("setup/config", config);
}

/**
 * Update onboarding step for resume capability.
 *
 * @param step - Current step identifier
 * @returns Confirmation message
 * @throws ApiError on server error
 */
export async function updateOnboardingStep(
  step: string,
): Promise<OnboardingResponse> {
  return authProxyPost<OnboardingResponse>("setup/onboarding-step", { step });
}

/**
 * Mark onboarding as completed.
 *
 * @returns Confirmation message
 * @throws ApiError on server error
 */
export async function completeOnboarding(): Promise<OnboardingResponse> {
  return authProxyPost<OnboardingResponse>("setup/complete-onboarding", {});
}

/**
 * Request a password reset token.
 *
 * In a local-first app, the token is returned in the response
 * (and printed to server logs) for the user to copy.
 *
 * @param email - User's email address
 * @returns Reset token and message
 * @throws ApiError on server error
 */
export async function requestPasswordReset(
  email: string,
): Promise<{ reset_token?: string; message: string }> {
  const resp = await apiPost<{ reset_token?: string; message: string }>(
    "/auth/password/reset",
    { email },
  );
  if (resp == null) {
    throw new ApiError(500, "EMPTY_RESPONSE", "Password reset request failed: no response from server");
  }
  return resp;
}

/**
 * Confirm password reset with a token.
 *
 * @param token - Reset token from email/console
 * @param newPassword - New password
 * @returns Confirmation message
 * @throws ApiError on invalid/expired token or server error
 */
export async function resetPassword(
  token: string,
  newPassword: string,
): Promise<MessageResponse> {
  const resp = await apiPost<MessageResponse>(
    "/auth/password/reset/confirm",
    { token, new_password: newPassword },
  );
  if (resp == null) {
    throw new ApiError(500, "EMPTY_RESPONSE", "Password reset failed: no response from server");
  }
  return resp;
}