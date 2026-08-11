/**
 * Generic API proxy Route Handler: /api/[...proxy]
 *
 * Forwards all /api/v1/* requests to the Go backend server-side.
 * This eliminates CORS issues and direct port access from the browser.
 *
 * More specific routes (e.g., /api/auth/[...proxy]) take priority in Next.js,
 * so auth flows with session cookie management are unaffected.
 *
 * Flow:
 * 1. Client calls apiPost("/auth/setup/status") → browser sends GET /api/v1/auth/setup/status
 * 2. This Route Handler intercepts → proxies to http://api:8080/api/v1/auth/setup/status
 * 3. Response returned to client — no CORS, no direct port access needed
 */

import { NextRequest, NextResponse } from "next/server";

/** Go backend base URL. */
const BACKEND_URL = process.env.BACKEND_URL ?? "http://localhost:8080";

/** Backend API prefix (matches Go router.go). */
const API_PREFIX = "/api/v1";

/**
 * Generic proxy handler — forwards any method to the Go backend.
 */
async function proxyRequest(
  request: NextRequest,
  proxyPath: string[],
): Promise<NextResponse> {
  const backendPath = `${API_PREFIX}/${proxyPath.join("/")}`;
  const url = new URL(backendPath, BACKEND_URL);

  // Copy query parameters
  request.nextUrl.searchParams.forEach((value, key) => {
    url.searchParams.set(key, value);
  });

  // Forward the request body as-is for non-GET/HEAD methods
  const init: RequestInit = {
    method: request.method,
    headers: new Headers(),
  };

  // Only attach body for methods that support it
  if (request.method !== "GET" && request.method !== "HEAD") {
    init.body = request.body;
    // Pass through the content-type header for streaming body
    const contentType = request.headers.get("content-type");
    if (contentType) {
      (init.headers as Headers).set("content-type", contentType);
    }
  }

  try {
    const backendResp = await fetch(url.toString(), init);

    // Forward response headers and status
    const responseHeaders = new Headers();
    backendResp.headers.forEach((value, key) => {
      // Skip headers that shouldn't be forwarded
      const lowerKey = key.toLowerCase();
      if (
        lowerKey === "transfer-encoding" ||
        lowerKey === "connection" ||
        lowerKey === "keep-alive"
      ) {
        return;
      }
      responseHeaders.set(key, value);
    });

    // Stream the response body back to the client
    return new NextResponse(backendResp.body, {
      status: backendResp.status,
      statusText: backendResp.statusText,
      headers: responseHeaders,
    });
  } catch (error) {
    console.error("[api/proxy] Backend request failed:", error);
    return NextResponse.json(
      { error: "Backend service unavailable" },
      { status: 502 },
    );
  }
}

// Export handlers for all HTTP methods
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ proxy: string[] }> },
) {
  const { proxy } = await params;
  return proxyRequest(request, proxy);
}

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ proxy: string[] }> },
) {
  const { proxy } = await params;
  return proxyRequest(request, proxy);
}

export async function PUT(
  request: NextRequest,
  { params }: { params: Promise<{ proxy: string[] }> },
) {
  const { proxy } = await params;
  return proxyRequest(request, proxy);
}

export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ proxy: string[] }> },
) {
  const { proxy } = await params;
  return proxyRequest(request, proxy);
}

export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ proxy: string[] }> },
) {
  const { proxy } = await params;
  return proxyRequest(request, proxy);
}
