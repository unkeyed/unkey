import { authMiddleware } from "@/lib/auth/middleware";
import { env } from "@/lib/env";
import { type NextRequest, NextResponse } from "next/server";
import type { NextFetchEvent } from "next/server";
import { SIGN_IN_URL } from "./lib/auth/types";

export default async function middleware(req: NextRequest, _evt: NextFetchEvent) {
  const url = new URL(req.url);

  // Special redirect for gateway.new
  if (url.host === "gateway.new") {
    return NextResponse.redirect("https://app.unkey.com/gateway-new");
  }

  const authProvider = env().AUTH_PROVIDER;
  const isEnabled = () => authProvider !== "local";

  // Define public paths that should bypass authentication
  const publicPaths = [
    "/auth/sign-in",
    "/auth/sign-up",
    "/auth/sso-callback",
    "/auth/oauth-sign-in",
    "/auth/join",
    "/favicon.ico",
    "/api/webhooks/stripe",
    "/api/v1/workos/webhooks",
    "/api/v1/github/verify",
    "/api/auth/refresh",
    "/_next",
  ];

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

  try {
    const { session, headers } = await authMiddleware(req);

    if (!session) {
      return NextResponse.redirect(new URL(SIGN_IN_URL, url));
    }

    return NextResponse.next({
      headers: headers,
    });
  } catch (error) {
    console.error("Middleware error:", error);
    return NextResponse.redirect(new URL(SIGN_IN_URL, url));
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
    "/gateways",
    "/gateways/(.*)",
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
    "/gateway-new",
    "/(api|trpc)(.*)",
    "/((?!.+\\.[\\w]+$|_next).*)",
    "/((?!_next/static|_next/image|images|favicon.ico|$).*)",
    "/robots.txt",
  ],
};
