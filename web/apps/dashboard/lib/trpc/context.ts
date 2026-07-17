import type { inferAsyncReturnType } from "@trpc/server";
import type { FetchCreateContextFnOptions } from "@trpc/server/adapters/fetch";
import type { NextRequest } from "next/server";

import { getAuth } from "../auth/get-auth";
import { getClientIp } from "../client-ip";
import { db } from "../db";

export async function createContext({ req }: FetchCreateContextFnOptions) {
  const authResult = await getAuth(req as NextRequest);
  const { userId, orgId } = authResult;

  let ws: Awaited<ReturnType<typeof db.query.workspaces.findFirst<{ with: { quotas: true } }>>> =
    undefined;

  // Only attempt workspace query if we have both userId and orgId
  // This prevents unnecessary queries during auth setup phase
  if (orgId && userId) {
    try {
      ws = await db.query.workspaces.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
        with: {
          quotas: true,
        },
      });
    } catch (error) {
      console.debug("Workspace query failed in context creation",
      });
      ws = undefined;
    }
  }

  return {
    req,
    audit: {
      userAgent: req.headers.get("user-agent") ?? undefined,
      // Recorded as `remote_ip` on every audit log, so it must be an address we trust rather than
      // whatever the client put in a forwarding header.
      location: getClientIp(req.headers) ?? "unknown",
    },
    user: authResult.userId
      ? {
          id: authResult.userId,
          // Profile from the sealed session cookie; saves provider API calls
          // for procedures that only need the signed-in user's profile.
          profile: authResult.user ?? null,
        }
      : null,
    workspace: ws,
    tenant: authResult.orgId
      ? {
          id: authResult.orgId,
          role: authResult.role,
        }
      : null,
  };
}

export type Context = inferAsyncReturnType<typeof createContext>;
