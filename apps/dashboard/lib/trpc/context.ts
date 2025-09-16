import type { inferAsyncReturnType } from "@trpc/server";
import type { FetchCreateContextFnOptions } from "@trpc/server/adapters/fetch";

import { getAuth } from "../auth/get-auth";
import { db } from "../db";

export async function createContext({ req }: FetchCreateContextFnOptions) {
  // biome-ignore lint/suspicious/noExplicitAny:This has to be generic so any is okay
  const { userId, orgId, role } = await getAuth(req as any);

  let ws: Awaited<ReturnType<typeof db.query.workspaces.findFirst>> = undefined;

  // Only attempt workspace query if we have both userId and orgId
  // This prevents unnecessary queries during auth setup phase
  if (orgId && userId) {
    try {
      ws = await db.query.workspaces.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
      });
    } catch (error) {
      // Log workspace query errors but don't fail context creation
      // This allows the frontend to handle workspace loading gracefully
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
    user: userId ? { id: userId } : null,
    workspace: ws,
    tenant: orgId
      ? {
          id: orgId,
          role,
        }
      : null,
  };
}

export type Context = inferAsyncReturnType<typeof createContext>;
