import { inferAsyncReturnType } from "@trpc/server";
import type { FetchCreateContextFnOptions } from "@trpc/server/adapters/fetch";

import { getAuth } from "@clerk/nextjs/server";

export async function createContext({ req }: FetchCreateContextFnOptions) {
  const { userId, orgId, orgRole } = getAuth(req as any);

  return {
    req,
    audit: {
      userAgent: req.headers.get("user-agent") ?? undefined,
      location: req.headers.get("x-forwarded-for") ?? process.env.VERCEL_REGION ?? "unknown",
    },
    user: userId ? { id: userId } : null,
    tenant:
      orgId && orgRole
        ? {
            id: orgId,
            role: orgRole,
          }
        : userId
          ? {
              id: userId,
              role: "owner",
            }
          : null,
  };
}

export type Context = inferAsyncReturnType<typeof createContext>;
