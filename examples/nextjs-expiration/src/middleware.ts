import { verifyKey } from "@unkey/api";
import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export async function middleware(request: NextRequest) {
  const authHeader = request.headers.get("Authorization");

  if (!authHeader) return new Response("Unauthorized", { status: 401 });

  const key = authHeader.split(" ")[1];

  if (!key) return new Response("Unauthorized", { status: 401 });

  const { result, error } = await verifyKey(authHeader);

  if (error) {
    return new Response(error.message, { status: 500 });
  }

  if (!result.valid) {
    return new Response("Unauthorized", { status: 401 });
  }

  const response = NextResponse.next();

  return response;
}

// See "Matching Paths" below to learn more
export const config = {
  matcher: "/api/:path*",
};
