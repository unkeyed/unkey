import type { inferAsyncReturnType } from "@trpc/server";
import type { FetchCreateContextFnOptions } from "@trpc/server/adapters/fetch";
import { parse } from "cookie"

import { serverAuth } from "../auth/server";

export async function createContext({ req }: FetchCreateContextFnOptions) {

  const cookies = req.headers.get("Cookie")
  const x = parse(cookies)

  const user = await serverAuth.getUserFromCookie(req.headers[""])


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
