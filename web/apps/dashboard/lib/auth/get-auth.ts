import { env } from "@/lib/env";
import { logOperation } from "@/lib/logging";
import { unstable_rethrow } from "next/navigation";
import type { NextRequest } from "next/server";
import { type WorkOSUserProfile, mapWorkOSUser } from "./map-workos-user";
import type { AuthenticatedUser, User } from "./types";
import { getWorkOSSession } from "./workos-session";

export type GetAuthResult = {
  userId: string | null;
  orgId: string | null;
  accessToken?: string;
  permissions?: readonly string[];
  role: string | null;
  // Profile embedded in the sealed session cookie, when available
  user?: User | null;
  impersonator?: {
    email: string;
    reason?: string | null;
  };
};

type AuthkitSession =
  | {
      sessionId: string;
      user: WorkOSUserProfile;
      organizationId?: string;
      role?: string;
      permissions?: string[];
      accessToken: string;
      impersonator?: {
        email: string;
        reason: string | null;
      };
    }
  | { user: null };

const ANONYMOUS: GetAuthResult = {
  userId: null,
  orgId: null,
  role: null,
  user: null,
};

export function mapAuthkitSession(session: AuthkitSession): GetAuthResult {
  if (!session.user) {
    return ANONYMOUS;
  }

  return {
    userId: session.user.id,
    orgId: session.organizationId ?? null,
    role: session.role ?? null,
    permissions: session.permissions,
    accessToken: session.accessToken,
    impersonator: session.impersonator,
    user: mapWorkOSUser(session.user),
  };
}

export function toAuthenticatedUser(auth: GetAuthResult): AuthenticatedUser | null {
  if (!auth.user) {
    return null;
  }
  return { ...auth.user, orgId: auth.orgId, role: auth.role };
}

function logAuthResolutionFailure(provider: "local" | "workos", error: unknown): void {
  console.error("Failed to resolve session", { provider, error });
  logOperation("warn", "Session resolution failed", {
    auth_event: "session_resolution",
    auth_outcome: "failure",
    auth_provider: provider,
  });
}

export async function getAuth(req?: NextRequest): Promise<GetAuthResult> {
  if (env().AUTH_PROVIDER === "local") {
    try {
      const { updateLocalSession } = await import("./sessions");
      const { session } = await updateLocalSession(req);

      return session ?? ANONYMOUS;
    } catch (error) {
      unstable_rethrow(error);
      logAuthResolutionFailure("local", error);
      return ANONYMOUS;
    }
  }

  try {
    return mapAuthkitSession(await getWorkOSSession());
  } catch (error) {
    unstable_rethrow(error);
    logAuthResolutionFailure("workos", error);
    return ANONYMOUS;
  }
}
