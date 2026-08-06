/**
 * Session utility — JWT encrypt/decrypt using jose (Edge-compatible).
 *
 * Used by:
 * - proxy.ts (formerly middleware.ts) to validate session cookies
 * - Route Handlers to create/verify sessions
 * - lib/dal.ts for Server Component auth checks
 *
 * Requires SESSION_SECRET env var (≥32 chars).
 *
 * @example
 *   import { encrypt, decrypt } from "@/lib/session";
 *   const token = await encrypt({ userId: "1", expiresAt: Date.now() + 86400 });
 *   const payload = await decrypt(token);
 */

import "server-only";
import { SignJWT, jwtVerify, type JWTPayload } from "jose";

function getSecretKey(): string {
  const secretKey = process.env.SESSION_SECRET;
  if (!secretKey) {
    // Fail closed — never sign cookies with a hard-coded fallback that could
    // be extracted from the bundle. SESSION_SECRET must be provided via env
    // in every environment.
    throw new Error(
      "SESSION_SECRET env var is required (≥32 chars). Set it in .env or your deployment environment.",
    );
  }
  // Warn (do not fail) when running outside production with a known dev value
  // so devs notice if they forgot to rotate the secret.
  if (
    process.env.NODE_ENV !== "production" &&
    secretKey === "dev-only-session-secret-change-in-production-32ch"
  ) {
    console.warn(
      "[session] SESSION_SECRET is set to the default dev value — change it before deploying",
    );
  }
  return secretKey;
}

/** Session payload stored in the JWT cookie. */
export interface SessionPayload extends JWTPayload {
  /** Backend access token (JWT from Go API). */
  accessToken: string;
  /** Backend refresh token. */
  refreshToken: string;
  /** Token expiry as Unix timestamp (seconds). */
  expiresAt: number;
}

/**
 * Encrypt a session payload into a signed JWT string.
 *
 * @param payload - Session data to encrypt
 * @returns Signed JWT string
 */
export async function encrypt(payload: SessionPayload): Promise<string> {
  return new SignJWT(payload as unknown as Record<string, unknown>)
    .setProtectedHeader({ alg: "HS256" })
    .setIssuedAt()
    .setExpirationTime("7d")
    .setIssuer("myjob-session")
    .sign(new TextEncoder().encode(getSecretKey()));
}

/**
 * Decrypt and verify a JWT session cookie.
 *
 * @param session - JWT string from cookie (may be undefined)
 * @returns Decoded payload, or null if invalid/expired
 */
export async function decrypt(
  session: string | undefined,
): Promise<SessionPayload | null> {
  if (!session) return null;

  try {
    const { payload } = await jwtVerify(session, new TextEncoder().encode(getSecretKey()), {
      algorithms: ["HS256"],
      issuer: "myjob-session",
    });
    return payload as unknown as SessionPayload;
  } catch {
    return null;
  }
}
