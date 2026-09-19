import { workosAuthEnv } from "@/lib/env";
import { cache } from "react";

export const AUTHKIT_MIDDLEWARE_HEADER = "x-workos-middleware";

export function hasAuthkitMiddleware(requestHeaders: Headers): boolean {
  return requestHeaders.has(AUTHKIT_MIDDLEWARE_HEADER);
}

export const getWorkOSSession = cache(async () => {
  workosAuthEnv();
  const { withAuth } = await import("@workos-inc/authkit-nextjs");
  return withAuth();
});
