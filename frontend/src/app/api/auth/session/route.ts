import { NextRequest, NextResponse } from "next/server";
import { decrypt } from "@/lib/session";

export async function GET(request: NextRequest) {
  const sessionCookie = request.cookies.get("session")?.value;
  const session = sessionCookie ? await decrypt(sessionCookie) : null;
  const authenticated = Boolean(
    session?.accessToken && (session.expiresAt ?? 0) > Math.floor(Date.now() / 1000)
  );
  return NextResponse.json({ authenticated });
}
