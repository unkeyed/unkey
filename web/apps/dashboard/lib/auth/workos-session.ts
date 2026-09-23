import { workosAuthEnv } from "@/lib/env";
import { headers } from "next/headers";
import { cache } from "react";

const AUTHKIT_MIDDLEWARE_HEADER = "x-workos-middleware";

export const getWorkOSSession = cache(async () => {
  workosAuthEnv();
  const { withAuth } = await import("@workos-inc/authkit-nextjs");
  return withAuth();
});

// AuthKit's middleware sets x-workos-middleware on every request it handles, and
// withAuth() throws on requests that did not pass through it.
export async function hasAuthkitMiddleware(): Promise<boolean> {
  return (await headers()).has(AUTHKIT_MIDDLEWARE_HEADER);
}
