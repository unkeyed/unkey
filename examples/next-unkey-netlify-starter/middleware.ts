import { verifyKey } from "@unkey/api";
import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export async function middleware(request: NextRequest) {
  const unkey = request.cookies.get("unkey-token")?.value;
  if (!unkey) {
    return NextResponse.redirect(new URL("/auth", request.url));
  }
  const { error, result } = await verifyKey(unkey);
  if (error) {
    return NextResponse.redirect(new URL("/auth", request.url));
  }
  const requestHeaders = new Headers(request.headers);
  const response = NextResponse.next({
    request: {
      headers: requestHeaders,
    },
  });
  response.headers.set("x-hello-from-unkey", JSON.stringify(result));
  return response;
}

// See "Matching Paths" below to learn more
export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - api (API routes)
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     */
    "/((?!api|_next/static|auth|_next/image|favicon.ico).*)",
  ],
};
