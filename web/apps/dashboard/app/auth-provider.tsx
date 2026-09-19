import { logAuthkitMiddlewareBypass } from "@/lib/auth/telemetry";
import { getWorkOSSession, hasAuthkitMiddleware } from "@/lib/auth/workos-session";
import { env } from "@/lib/env";
import { headers } from "next/headers";
import { notFound } from "next/navigation";
import type React from "react";

export async function AuthProvider({ children }: { children: React.ReactNode }) {
  if (env().AUTH_PROVIDER === "local") {
    return children;
  }

  // The AuthKit middleware matcher skips paths that look like static files, so a
  // request for an asset that does not exist falls through to a page route
  // without the middleware request headers that withAuth requires. Such paths
  // are not application routes.
  const requestHeaders = await headers();
  if (!hasAuthkitMiddleware(requestHeaders)) {
    logAuthkitMiddlewareBypass(requestHeaders);
    notFound();
  }

  const [{ AuthKitProvider, Impersonation }, session] = await Promise.all([
    import("@workos-inc/authkit-nextjs/components"),
    getWorkOSSession(),
  ]);
  const { accessToken: _accessToken, ...initialAuth } = session;

  return (
    <AuthKitProvider initialAuth={initialAuth}>
      {children}
      <Impersonation side="top" returnTo="/auth/sign-in" />
    </AuthKitProvider>
  );
}
