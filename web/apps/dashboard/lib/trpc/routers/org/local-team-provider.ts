import { type LocalAuthProvider, localAuth } from "@/lib/auth/local";
import { env } from "@/lib/env";
import { TRPCError } from "@trpc/server";

/**
 * Custom team-management procedures exist only for deterministic local mode.
 * WorkOS mode uses the managed User Management widget directly.
 */
export function getLocalTeamProvider(): LocalAuthProvider {
  if (env().AUTH_PROVIDER !== "local") {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "Team management is handled by WorkOS",
    });
  }

  return localAuth;
}
