import { workosAuthEnv } from "@/lib/env";
import { logOperation } from "@/lib/logging";
import type { NoUserInfo, UserInfo } from "@workos-inc/authkit-nextjs";
import { cookies, headers } from "next/headers";
import { cache } from "react";

/** Sentinel AuthKit's middleware sets on every request it handles. */
const AUTHKIT_MIDDLEWARE_HEADER = "x-workos-middleware";

const ANONYMOUS_SESSION: NoUserInfo = { user: null };

/**
 * Next renders the root layout -- and with it the AuthKit provider -- for any
 * request that matches no route, including probes for paths the middleware
 * matcher deliberately skips (anything that looks like a static file). Those
 * renders reach `withAuth` with no middleware headers, where it throws.
 *
 * Treating them as signed out turns a crashed 404 back into a plain 404, and
 * it fails closed: without a session, every protected page still redirects to
 * sign-in.
 */
export const getWorkOSSession = cache(async (): Promise<UserInfo | NoUserInfo> => {
  const { WORKOS_COOKIE_NAME } = workosAuthEnv();

  if (!(await headers()).get(AUTHKIT_MIDDLEWARE_HEADER)) {
    await reportRenderWithoutMiddleware(WORKOS_COOKIE_NAME);
    return ANONYMOUS_SESSION;
  }

  const { withAuth } = await import("@workos-inc/authkit-nextjs");
  return withAuth();
});

/**
 * A caller with no session cookie had no session to lose, so an uncovered
 * render is the expected outcome and not worth reporting. A caller that did
 * carry one was just downgraded to anonymous, which only a matcher that stopped
 * covering a real route can cause.
 */
async function reportRenderWithoutMiddleware(cookieName: string): Promise<void> {
  const carriedSession = (await cookies()).has(cookieName);

  logOperation(carriedSession ? "warn" : "debug", "Rendered without AuthKit middleware headers", {
    auth_event: "session_resolution",
    auth_outcome: "failure",
    auth_carried_session: carriedSession,
  });
}
