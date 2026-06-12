import { authMiddleware } from "@/lib/auth/middleware";
import { env } from "@/lib/env";
import { type NextRequest, NextResponse } from "next/server";
import type { NextFetchEvent } from "next/server";
import { SIGN_IN_URL } from "./lib/auth/types";

// biome-ignore lint/style/noDefaultExport: required by next.js
export default async function proxy(req: NextRequest, _evt: NextFetchEvent) {
  const url = new URL(req.url);

  // Special redirect for sentinel.new
  if (url.host === "sentinel.new") {
    return NextResponse.redirect("https://app.unkey.com/sentinel-new");
  }

  // Redirect /auth/join to /join to bypass auth layout
  if (url.pathname === "/auth/join") {
    const joinUrl = new URL("/join", url);
    // Preserve all query parameters (including invitation_token)
    joinUrl.search = url.search;
    return NextResponse.redirect(joinUrl, 307);
  }

  const AUTH_PROVIDER = env().AUTH_PROVIDER;
  const isEnabled = () => AUTH_PROVIDER !== "local";

  // Define public paths that should bypass authentication
  const publicPaths = [
    "/auth/sign-in",
    "/auth/sign-up",
    "/auth/sso-callback",
    "/auth/oauth-sign-in",
    "/auth/join",
    "/join",
    "/join/success",
    "/favicon.ico",
    "/api/webhooks/stripe",
    "/api/webhooks/workos",
    "/api/v1/github/verify",
    "/api/auth/refresh",
    "/success",
    "/_next/*",
    // /integrations/github/callback is intentionally NOT public: the page
    // calls trpc.github.registerInstallation under the user's session, and
    // marking it public would let an unauthenticated visitor reach the page
    // with a phishing-friendly URL. The page now requires an authenticated
    // session and verifies a server-signed `state` HMAC.
    "/integrations/domain-connect/callback",
  ];

  // Signed-in users get bounced away from the sign-in/sign-up pages. This
  // used to live in the auth layout, but setting the session cookie in a
  // server action re-renders that layout mid-flow, so its redirect raced
  // ahead of the action's own navigation (e.g. the invite flow's
  // /join/success) and flashed the dashboard. Only document GETs are
  // bounced here; server-action POSTs pass through untouched.
  const isAuthEntryPath =
    url.pathname.startsWith("/auth/sign-in") || url.pathname.startsWith("/auth/sign-up");
  if (isAuthEntryPath && req.method === "GET" && isEnabled()) {
    try {
      const { session } = await authMiddleware(req);
      if (session) {
        return NextResponse.redirect(new URL("/apis", url));
      }
    } catch (_error) {
      // Fall through to render the auth page
    }
  }

  // Check if the current path is in the public paths list
  const isPublicPath = (path: string) => {
    return publicPaths.some((pubPath) => {
      // Exact match
      if (pubPath === path) {
        return true;
      }
      // Path starts with pubPath (for directory matches like /_next/*)
      if (pubPath.endsWith("*") && path.startsWith(pubPath.slice(0, -1))) {
        return true;
      }

      return false;
    });
  };

  // Skip authentication for public paths
  if (isPublicPath(url.pathname)) {
    return NextResponse.next();
  }

  // Only run auth middleware if auth is enabled
  if (!isEnabled()) {
    return NextResponse.next();
  }

  // API routes are fetched programmatically, so redirecting them to the
  // sign-in page hands an HTML document to a JSON client (tRPC throws
  // `Unexpected token '<'` when a query fires after the session ended).
  // Every route under /api enforces its own auth and returns a JSON 401
  // (tRPC via requireUser, the rest explicitly), so let them through and
  // fail with a proper JSON error instead.
  const isApiPath = url.pathname.startsWith("/api/");

  try {
    const { session, headers } = await authMiddleware(req);

    if (!session) {
      if (isApiPath) {
        return NextResponse.next();
      }
      const signInUrl = new URL(SIGN_IN_URL, url);
      const currentPath = url.pathname + url.search;
      if (currentPath && currentPath !== "/") {
        signInUrl.searchParams.set("redirect", currentPath);
      }
      return NextResponse.redirect(signInUrl);
    }

    // Custom headers (session, x-current-path) must be added to the *request*
    // headers so server components see them via `headers()`. Setting them as
    // response headers via `NextResponse.next({ headers })` corrupts Next.js
    // 16's router state parsing during soft navigation.
    const requestHeaders = new Headers(req.headers);
    headers.forEach((value, key) => {
      if (key.toLowerCase() === "set-cookie") {
        return;
      }
      requestHeaders.set(key, value);
    });
    requestHeaders.set("x-current-path", url.pathname + url.search);

    const response = NextResponse.next({
      request: { headers: requestHeaders },
    });

    // Set-Cookie must remain on the response, not be forwarded to the request.
    const setCookies = headers.getSetCookie?.() ?? [];
    for (const cookie of setCookies) {
      response.headers.append("Set-Cookie", cookie);
    }

    return response;
  } catch (error) {
    console.error("Middleware error:", error);
    if (isApiPath) {
      return NextResponse.next();
    }
    const signInUrl = new URL(SIGN_IN_URL, url);
    const currentPath = url.pathname + url.search;
    if (currentPath && currentPath !== "/") {
      signInUrl.searchParams.set("redirect", currentPath);
    }
    return NextResponse.redirect(signInUrl);
  }
}

export const config = {
  matcher: [
    "/",
    "/apis",
    "/apis/(.*)",
    "/audit",
    "/audit/(.*)",
    "/authorization",
    "/authorization/(.*)",
    "/debug",
    "/debug/(.*)",
    "/sentinels",
    "/sentinels/(.*)",
    "/new",
    "/new(.*)",
    "/overview",
    "/overview/(.*)",
    "/ratelimits",
    "/ratelimits/(.*)",
    "/secrets",
    "/secrets/(.*)",
    "/semant-cache",
    "/semantic-cache/(.*)",
    "/settings",
    "/settings/(.*)",
    "/success",
    "/success/(.*)",
    "/auth/(.*)",
    "/sentinel-new",
    "/(api|trpc)(.*)",
    "/((?!.+\\.[\\w]+$|_next).*)",
    "/((?!_next/static|_next/image|images|favicon.ico|$).*)",
    "/robots.txt",
  ],
};
