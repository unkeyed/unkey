import { inferAsyncReturnType } from "@trpc/server";
import * as trpcNext from "@trpc/server/adapters/next";

import { getAuth } from "@clerk/nextjs/server";
export async function createContext({ req, res }: trpcNext.CreateNextContextOptions) {
  const { userId, orgId, orgSlug, orgRole } = getAuth(req);

  return {
    req,
    res,
    user: userId ? { id: userId } : null,
    tenant:
      orgId && orgSlug && orgRole
        ? {
            id: orgId,
            slug: orgSlug,
            role: orgRole,
          }
        : userId
        ? {
            id: userId,
            slug: "home",
            role: "owner",
          }
        : null,
  };
}

export type Context = inferAsyncReturnType<typeof createContext>;
