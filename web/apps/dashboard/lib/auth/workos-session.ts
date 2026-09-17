import { logManagedAuthOutcome } from "@/lib/auth/telemetry";
import { workosAuthEnv } from "@/lib/env";
import { cache } from "react";

const UNCOVERED_MIDDLEWARE_ERROR = "isn't covered by the AuthKit middleware";
const UNCOVERED_MIDDLEWARE_ROUTE = /on (.+?) that isn't covered by the AuthKit middleware/;

export const getWorkOSSession = cache(async () => {
  workosAuthEnv();
  const { withAuth } = await import("@workos-inc/authkit-nextjs");

  try {
    return await withAuth();
  } catch (error) {
    // The root provider renders for every path the app router matches, but the
    // proxy's matcher deliberately excludes paths with a file extension, so asset
    // probes (e.g. /apple-touch-icon-precomposed.png) reach the provider without
    // AuthKit's request headers. Those requests carry no session.
    if (error instanceof Error && error.message.includes(UNCOVERED_MIDDLEWARE_ERROR)) {
      logManagedAuthOutcome(
        "uncovered_route",
        "failure",
        UNCOVERED_MIDDLEWARE_ROUTE.exec(error.message)?.[1],
      );
      return { user: null };
    }
    throw error;
  }
});
