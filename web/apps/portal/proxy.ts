import { type NextRequest, NextResponse } from "next/server";

const SESSION_COOKIE_NAME = "portal_session";

/**
 * Middleware that protects all portal routes except the root page (session exchange entry point).
 * If no valid session cookie exists, redirects to the root page with an error.
 */
export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Allow the root page (session exchange entry point) through without a cookie.
  // With basePath="/portal", the root page path is just "/".
  if (pathname === "/" || pathname === "") {
    return NextResponse.next();
  }

  const sessionToken = request.cookies.get(SESSION_COOKIE_NAME)?.value;

  if (!sessionToken) {
    // No session cookie — redirect to root with error indicator
    const url = request.nextUrl.clone();
    url.pathname = "/";
    url.searchParams.delete("session");
    return NextResponse.redirect(url);
  }

  // Session cookie exists — allow the request through.
  // The actual session validation (expiry check) happens at the Unkey API layer
  // when the browser makes direct API calls with the session token.
  return NextResponse.next();
}

export const config = {
  // Match all routes except: root page (session exchange), static files, Next.js internals
  matcher: ["/((?!_next/static|_next/image|favicon.ico).+)"],
};
