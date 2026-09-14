import { expireLegacySession } from "@/lib/auth/legacy-session";
import { sanitizeRedirectPath } from "@/lib/auth/redirect-utils";
import { logManagedAuthOutcome } from "@/lib/auth/telemetry";
import { env, workosAuthEnv } from "@/lib/env";
import { getBaseUrl } from "@/lib/utils";
import { NextRequest, NextResponse } from "next/server";

const PUBLIC_PATHS = [
  "/auth/sso-callback",
  "/auth/error",
  "/favicon.ico",
  "/api/webhooks/stripe",
  "/api/webhooks/workos",
  "/api/v1/github/verify",
  "/success",
  "/share",
  "/_next/*",
  "/integrations/domain-connect/callback",
] as const;

const PUBLIC_ROUTE_HANDLER_PATHS = [
  "/auth/sso-callback",
  "/join",
  "/api/webhooks/stripe",
  "/api/webhooks/workos",
  "/api/v1/github/verify",
] as const;

function isPublicPath(path: string): boolean {
  return PUBLIC_PATHS.some((publicPath) =>
    publicPath.endsWith("*") ? path.startsWith(publicPath.slice(0, -1)) : path === publicPath,
  );
}

function isPublicRouteHandler(path: string): boolean {
  return PUBLIC_ROUTE_HANDLER_PATHS.some(
    (routePath) => path === routePath || path.startsWith(`${routePath}/`),
  );
}

function getWorkosRedirectUri(requestUrl: URL, vercelUrl: string | undefined): string {
  const configuredBaseUrl = new URL(getBaseUrl());
  const trustedOrigins = new Set([configuredBaseUrl.origin]);
  if (vercelUrl) {
    trustedOrigins.add(new URL(`https://${vercelUrl}`).origin);
  }

  const callbackOrigin = trustedOrigins.has(requestUrl.origin)
    ? requestUrl.origin
    : configuredBaseUrl.origin;
  return new URL("/auth/sso-callback", callbackOrigin).toString();
}

// biome-ignore lint/style/noDefaultExport: required by Next.js
export default async function proxy(req: NextRequest) {
  const url = req.nextUrl;
  const environment = env();

  // Preserve the retired domain and route for inbound compatibility.
  if (url.host === "sentinel.new") {
    return NextResponse.redirect("https://app.unkey.com/sentinel-new");
  }

  if (environment.AUTH_PROVIDER === "local") {
    if (url.pathname === "/auth/join") {
      const joinUrl = new URL("/join", url);
      joinUrl.search = url.search;
      return NextResponse.redirect(joinUrl, 307);
    }
    return NextResponse.next();
  }

  // Route handlers do not render the root AuthKitProvider, so they can bypass
  // session middleware. Public pages still need the middleware-owned request
  // headers consumed by withAuth in the root provider.
  if (isPublicRouteHandler(url.pathname)) {
    return NextResponse.next();
  }

  workosAuthEnv();
  const { authkit, handleAuthkitHeaders } = await import("@workos-inc/authkit-nextjs");
  const isAuthEntry =
    req.method === "GET" &&
    (url.pathname.startsWith("/auth/sign-in") || url.pathname.startsWith("/auth/sign-up"));
  const returnPath = sanitizeRedirectPath(url.searchParams.get("redirect"));
  const authkitRequest = isAuthEntry
    ? new NextRequest(new URL(returnPath, url), {
        method: req.method,
        headers: req.headers,
      })
    : req;
  const redirectUri = getWorkosRedirectUri(url, environment.VERCEL_URL);
  const { session, headers, authorizationUrl } = await authkit(authkitRequest, {
    redirectUri,
    screenHint: url.pathname.startsWith("/auth/sign-up") ? "sign-up" : "sign-in",
    onSessionRefreshSuccess: () => {
      logManagedAuthOutcome("session_refresh", "success");
    },
    onSessionRefreshError: () => {
      logManagedAuthOutcome("session_refresh", "failure");
    },
  });

  if (isAuthEntry) {
    if (session.user) {
      return expireLegacySession(
        req,
        handleAuthkitHeaders(req, headers, {
          redirect: returnPath,
        }),
      );
    }

    if (!authorizationUrl) {
      return expireLegacySession(
        req,
        handleAuthkitHeaders(req, headers, { redirect: "/auth/error?reason=entry" }),
      );
    }

    return expireLegacySession(
      req,
      handleAuthkitHeaders(req, headers, { redirect: authorizationUrl }),
    );
  }

  const isApiPath = url.pathname.startsWith("/api/");
  if (!session.user && !isApiPath && !isPublicPath(url.pathname)) {
    return expireLegacySession(
      req,
      handleAuthkitHeaders(req, headers, {
        redirect: authorizationUrl ?? "/auth/error?reason=session",
      }),
    );
  }

  return expireLegacySession(req, handleAuthkitHeaders(req, headers));
}

export const config = {
  matcher: [
    // API paths must always receive AuthKit's trusted request headers. tRPC
    // procedure names contain dots and otherwise look like static file paths.
    "/api/:path*",
    "/((?!_next/static|_next/image|images|favicon.ico|.+\\.[\\w]+$).*)",
  ],
};
