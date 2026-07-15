import type { inferAsyncReturnType } from "@trpc/server";
import type { FetchCreateContextFnOptions } from "@trpc/server/adapters/fetch";
import type { NextRequest } from "next/server";

import { getAuth } from "../auth/get-auth";
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
      // Log workspace query errors but don't fail context creation.
      // Procedures that require a workspace decide how to react.
      console.debug("Workspace query failed in context creation:", {
        orgId,
        userId,
        error: error instanceof Error ? error.message : String(error),
      });
      ws = undefined;
    }
  }

  return {
    req,
    audit: {
      userAgent: req.headers.get("user-agent") ?? undefined,
      location: req.headers.get("x-forwarded-for") ?? process.env.VERCEL_REGION ?? "unknown",
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
