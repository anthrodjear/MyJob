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
    if (process.env.NODE_ENV === "production") {
      throw new Error("SESSION_SECRET env var is required in production (≥32 chars)");
    }
    console.warn("[session] Using fallback SESSION_SECRET — set SESSION_SECRET in production");
    return "dev-fallback-secret-change-in-production-min-32-chars";
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
