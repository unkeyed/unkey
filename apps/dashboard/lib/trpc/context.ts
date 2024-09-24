import type { inferAsyncReturnType } from "@trpc/server";
import type { CreateNextContextOptions } from "@trpc/server/adapters/next";

import { serverAuth } from "../auth/server";

export async function createContext({ req }: CreateNextContextOptions) {
  const user = await serverAuth.getUser()

  return {
    req,
    audit: {
      userAgent: req.headers.get("user-agent") ?? undefined,
      location: req.headers.get("x-forwarded-for") ?? process.env.VERCEL_REGION ?? "unknown",
    },
    user: user ? { id: user.id } : null,
    tenant: user ? { id: user.id } : null,
  };
}

export type Context = inferAsyncReturnType<typeof createContext>;
