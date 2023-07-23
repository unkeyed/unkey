import { inferAsyncReturnType } from "@trpc/server";
import * as trpcFetch from "@trpc/server/adapters/fetch";

import { getAuth } from "@clerk/nextjs/server";
export async function createContext({ req }: trpcFetch.FetchCreateContextFnOptions) {
  // rome-ignore lint/suspicious/noExplicitAny: TODO
  const { userId, orgId, orgSlug, orgRole } = getAuth(req as any);

  return {
    req,
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
