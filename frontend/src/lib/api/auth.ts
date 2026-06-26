/**
 * Auth API functions — login and password management.
 *
 * Backend endpoints:
 * - POST /auth/login → { access_token, expires_at }
 * - POST /auth/change-password → { message }
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
}

/** Response from POST /auth/setup. */
export interface SetupResponse {
  message: string;
}

/**
 * Check if setup is required (no users exist).
 *
 * @returns Setup status with setup_required flag
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
