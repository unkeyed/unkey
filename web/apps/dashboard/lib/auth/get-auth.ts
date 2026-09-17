import { env } from "@/lib/env";
import type { NextRequest } from "next/server";
import { type WorkOSUserProfile, mapWorkOSUser } from "./map-workos-user";
import type { User } from "./types";
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

export function mapAuthkitSession(session: AuthkitSession): GetAuthResult {
  if (!session.user) {
    return {
      userId: null,
      orgId: null,
      role: null,
      user: null,
    };
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

export async function getAuth(req?: NextRequest): Promise<GetAuthResult> {
  if (env().AUTH_PROVIDER === "local") {
    try {
      const { updateLocalSession } = await import("./sessions");
      const { session } = await updateLocalSession(req);

      return (
        session ?? {
          userId: null,
          orgId: null,
          role: null,
          user: null,
        }
      );
    } catch {
      return {
        userId: null,
        orgId: null,
        role: null,
        user: null,
      };
    }
  }

  try {
    return mapAuthkitSession(await getWorkOSSession());
  } catch (_error) {
    return {
      userId: null,
      orgId: null,
      role: null,
      user: null,
    };
  }
}
