import type { inferAsyncReturnType } from "@trpc/server";
import type { FetchCreateContextFnOptions } from "@trpc/server/adapters/fetch";

import { getAuth } from "@/lib/auth/get-auth";
import { db } from "../db";

export async function createContext({ req }: FetchCreateContextFnOptions) {
  const { userId, orgId, orgRole } = await getAuth(req as any);

  const ws = orgId
    ? await db.query.workspaces.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
      })
    : undefined;

  return {
    req,
    audit: {
      userAgent: req.headers.get("user-agent") ?? undefined,
      location: req.headers.get("x-forwarded-for") ?? process.env.VERCEL_REGION ?? "unknown",
    },
    user: userId ? { id: userId } : null,
    workspace: ws,
    tenant:
      orgId && orgRole
        ? {
            id: orgId,
            role: orgRole,
          }
        : null,
  };
}

export type Context = inferAsyncReturnType<typeof createContext>;
