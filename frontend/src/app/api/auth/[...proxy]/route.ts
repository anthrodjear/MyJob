/**
 * Route Handler: /api/auth/[...proxy]
 *
 * Proxies auth requests to the Go backend and manages the HTTP-only
 * session cookie. This is the SERVER-SIDE auth entry point.
 *
 * Flow:
 * 1. Login: password → Go backend → JWT tokens → encrypt into session cookie
 * 2. Refresh: session cookie → refresh token → Go backend → new tokens → update cookie
 * 3. Logout: clear session cookie
 *
 * The session cookie contains the Go backend's access/refresh tokens encrypted
 * with jose (HS256). Server Components read this cookie via cookies() and
 * forward the access token to the Go backend as Authorization header.
 *
 * @see lib/session.ts — JWT encrypt/decrypt
 * @see proxy.ts — client-side redirect layer (reads this cookie)
 * @see lib/dal.ts — Server Component auth check (reads this cookie)
 */

import { NextRequest, NextResponse } from "next/server";
import { cookies } from "next/headers";
import { encrypt, decrypt } from "@/lib/session";

/** Go backend base URL. */
const BACKEND_URL = process.env.BACKEND_URL ?? "http://localhost:8080";

/** Session cookie name. */
const SESSION_COOKIE = "session";

/** Backend API prefix (matches Go router.go). */
const API_PREFIX = "/api/v1";

/**
 * Proxy login to Go backend and set session cookie on success.
 *
 * POST /api/auth/login → POST /api/v1/auth/login → set HTTP-only cookie
 */
async function handleLogin(request: NextRequest): Promise<NextResponse> {
  let body: Record<string, unknown>;
  try {
    body = await request.json();
  } catch {
    return NextResponse.json({ error: "Invalid JSON" }, { status: 400 });
  }

  if (typeof body?.password !== "string" || body.password.length === 0) {
    return NextResponse.json(
      { error: "Password is required" },
      { status: 400 },
    );
  }

  // Call Go backend login
  const backendResp = await fetch(`${BACKEND_URL}${API_PREFIX}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password: body.password }),
  });

  if (!backendResp.ok) {
    // Log server-side, don't leak backend internals to client
    console.error("[auth/proxy] Login failed:", backendResp.status);
    return NextResponse.json(
      { error: "Login failed" },
      { status: backendResp.status },
    );
  }

  const loginData = await backendResp.json();

  if (typeof loginData?.access_token !== "string") {
    return NextResponse.json(
      { error: "Invalid login response from server" },
      { status: 500 },
    );
  }

  // Encrypt tokens into session cookie
  const sessionToken = await encrypt({
    accessToken: loginData.access_token,
    refreshToken: loginData.refresh_token ?? "",
    expiresAt: loginData.expires_at ?? 0,
  });

  // Set HTTP-only secure cookie
  const cookieStore = await cookies();
  cookieStore.set(SESSION_COOKIE, sessionToken, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: 7 * 24 * 60 * 60, // 7 days
  });

  return NextResponse.json({
    success: true,
    expires_at: loginData.expires_at,
  });
}

/**
 * Proxy token refresh to Go backend and update session cookie.
 *
 * POST /api/auth/refresh → read session cookie → POST /api/v1/auth/refresh → update cookie
 */
async function handleRefresh(request: NextRequest): Promise<NextResponse> {
  // Read current session cookie
  const sessionCookie = request.cookies.get(SESSION_COOKIE)?.value;
  const session = await decrypt(sessionCookie);

  if (!session?.refreshToken) {
    return NextResponse.json(
      { error: "No refresh token available" },
      { status: 401 },
    );
  }

  // Call Go backend refresh
  const backendResp = await fetch(`${BACKEND_URL}${API_PREFIX}/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: session.refreshToken }),
  });

  if (!backendResp.ok) {
    // Refresh token expired — clear cookie
    const response = NextResponse.json(
      { error: "Refresh failed" },
      { status: 401 },
    );
    response.cookies.delete(SESSION_COOKIE);
    return response;
  }

  const refreshData = await backendResp.json();

  if (typeof refreshData?.access_token !== "string") {
    return NextResponse.json(
      { error: "Invalid refresh response from server" },
      { status: 500 },
    );
  }

  // Encrypt new tokens into session cookie
  const sessionToken = await encrypt({
    accessToken: refreshData.access_token,
    refreshToken: refreshData.refresh_token ?? session.refreshToken,
    expiresAt: refreshData.expires_at ?? 0,
  });

  const response = NextResponse.json({
    success: true,
    expires_at: refreshData.expires_at,
  });

  // Update HTTP-only cookie with new session
  response.cookies.set(SESSION_COOKIE, sessionToken, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: 7 * 24 * 60 * 60,
  });

  return response;
}

/**
 * Logout — revoke refresh token on Go backend and clear session cookie.
 *
 * POST /api/auth/logout → revoke on backend → clear cookie
 */
async function handleLogout(request: NextRequest): Promise<NextResponse> {
  // Read session cookie to get refresh token for backend revocation
  const sessionCookie = request.cookies.get(SESSION_COOKIE)?.value;
  const session = await decrypt(sessionCookie);

  // Best-effort backend revocation — cookie is cleared regardless
  if (session?.refreshToken) {
    await fetch(`${BACKEND_URL}${API_PREFIX}/auth/logout`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: session.refreshToken }),
    }).catch(() => {
      /* ignore — cookie is cleared regardless */
    });
  }

  const response = NextResponse.json({ success: true });
  response.cookies.delete(SESSION_COOKIE);
  return response;
}

/**
 * Route handler dispatcher — routes by path within /api/auth/*.
 */
export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ proxy: string[] }> },
) {
  const { proxy } = await params;
  const action = proxy?.[0];

  switch (action) {
    case "login":
      return handleLogin(request);
    case "refresh":
      return handleRefresh(request);
    case "logout":
      return handleLogout(request);
    default:
      return NextResponse.json(
        { error: `Unknown auth action: ${action}` },
        { status: 404 },
      );
  }
}
