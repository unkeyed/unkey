import type { inferAsyncReturnType } from "@trpc/server";
import type { FetchCreateContextFnOptions } from "@trpc/server/adapters/fetch";
import type { NextRequest } from "next/server";

import { getAuth } from "../auth/get-auth";
import { getClientIp } from "../client-ip";
import { loadWorkspace } from "../db/load-workspace";

export async function createContext({ req }: FetchCreateContextFnOptions) {
  const authResult = await getAuth(req as NextRequest);
  const { userId, orgId } = authResult;

  let workspace: Awaited<ReturnType<typeof loadWorkspace>>;
  if (orgId && userId) {
    try {
      workspace = await loadWorkspace(orgId);
    } catch (_error) {
      console.debug("Workspace query failed in context creation");
    }
  }

  return {
    req,
    audit: {
      userAgent: req.headers.get("user-agent") ?? undefined,
      // Recorded as `remote_ip` on every audit log, so that value must come from a trusted header.
      location: getClientIp(req.headers) ?? "unknown",
    },
    user: authResult.userId
      ? {
          id: authResult.userId,
          // The sealed session profile avoids provider API calls for profile-only procedures.
          profile: authResult.user ?? null,
        }
      : null,
    workspace,
    tenant: authResult.orgId
      ? {
          id: authResult.orgId,
          role: authResult.role,
        }
      : null,
  };
}

export type Context = inferAsyncReturnType<typeof createContext>;
