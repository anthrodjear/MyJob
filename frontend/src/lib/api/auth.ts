/**
 * Auth API functions — login, password management, and onboarding.
 *
 * Backend endpoints:
 * - POST /auth/login → { access_token, expires_at }
 * - POST /auth/change-password → { message }
 * - GET /auth/setup/status → { setup_required, step }
 * - POST /auth/setup → { message }
 * - POST /auth/setup/test-llm → { valid }
 * - POST /auth/setup/test-voice → { valid }
 * - POST /auth/setup/test-email → { valid }
 * - POST /auth/setup/config → { message }
 * - POST /auth/setup/onboarding-step → { message }
 * - POST /auth/setup/complete-onboarding → { message }
 *
 * Token storage is handled by lib/auth.ts (localStorage).
 * The API client (lib/api/client.ts) automatically injects the token.
 *
 * @example
 *   import { login } from "@/lib/api/auth";
 *   const resp = await login("my-password");
 *   // resp.access_token is stored, all subsequent apiFetch calls include it
 */

import { apiGet, apiPost } from "./client";
import { setAuthToken } from "@/lib/auth";

/** Response from POST /auth/login. */
export interface LoginResponse {
  access_token: string;
  expires_at: number;
}

/** Response from POST /auth/change-password. */
export interface ChangePasswordResponse {
  message: string;
}

/**
 * Authenticate with the backend and store the JWT.
 *
 * @param password - User's password (single-user local app)
 * @returns Login response with access_token and expiry
 * @throws ApiError on invalid credentials or server error
 *
 * @example
 *   const { access_token } = await login("my-password");
 *   // Token is now stored in localStorage — all apiFetch calls use it
 */
export async function login(password: string): Promise<LoginResponse> {
  const resp = await apiPost<LoginResponse>("auth/login", { password });
  if (resp == null) {
    throw new Error("Login failed: no response from server");
  }
  // Store token for subsequent API calls
  setAuthToken(resp.access_token);
  return resp;
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
): Promise<ChangePasswordResponse> {
  const resp = await apiPost<ChangePasswordResponse>("auth/change-password", {
    current_password: currentPassword,
    new_password: newPassword,
  });
  if (resp == null) {
    throw new Error("Password change failed: no response from server");
  }
  return resp;
}

// --- Setup API ---

/** Response from GET /auth/setup/status. */
export interface SetupStatusResponse {
  setup_required: boolean;
  step?: string;
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
  const resp = await apiGet<SetupStatusResponse>("auth/setup/status");
  if (resp == null) {
    throw new Error("Setup status check failed: no response from server");
  }
  return resp;
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
  const resp = await apiPost<SetupResponse>("auth/setup", {
    username,
    email,
    password,
  });
  if (resp == null) {
    throw new Error("Setup failed: no response from server");
  }
  return resp;
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
  const resp = await apiPost<TestServiceResponse>("auth/setup/test-llm", {
    provider,
    api_key: apiKey,
  });
  if (resp == null) {
    throw new Error("LLM test failed: no response from server");
  }
  return resp;
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
  const resp = await apiPost<TestServiceResponse>("auth/setup/test-voice", {
    url,
    api_key: apiKey,
    api_secret: apiSecret,
  });
  if (resp == null) {
    throw new Error("Voice test failed: no response from server");
  }
  return resp;
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
  const resp = await apiPost<TestServiceResponse>("auth/setup/test-email", {
    tenant_id: tenantId,
    client_id: clientId,
    client_secret: clientSecret,
  });
  if (resp == null) {
    throw new Error("Email test failed: no response from server");
  }
  return resp;
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
  const resp = await apiPost<OnboardingResponse>("auth/setup/config", config);
  if (resp == null) {
    throw new Error("Config save failed: no response from server");
  }
  return resp;
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
  const resp = await apiPost<OnboardingResponse>(
    "auth/setup/onboarding-step",
    { step },
  );
  if (resp == null) {
    throw new Error("Step update failed: no response from server");
  }
  return resp;
}

/**
 * Mark onboarding as completed.
 *
 * @returns Confirmation message
 * @throws ApiError on server error
 */
export async function completeOnboarding(): Promise<OnboardingResponse> {
  const resp = await apiPost<OnboardingResponse>(
    "auth/setup/complete-onboarding",
    {},
  );
  if (resp == null) {
    throw new Error("Onboarding complete failed: no response from server");
  }
  return resp;
}
