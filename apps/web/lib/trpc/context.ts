import { TRPCError, inferAsyncReturnType } from "@trpc/server";
import type { FetchCreateContextFnOptions } from "@trpc/server/adapters/fetch";

import { lucia } from "../auth";
export async function createContext({ req, resHeaders }: FetchCreateContextFnOptions) {
  console.log(req);
  console.log(lucia.sessionCookieName);

  const cookies = req.headers.get("cookie");
  const sessionId = cookies
    ?.split(";")
    .find((c) => {
      return c.trim().split("=").at(0) === lucia.sessionCookieName;
    })
    ?.split("=")
    .at(-1);
  console.log({ sessionId });
  if (!sessionId) {
    // resHeaders.append("Set-Cookie", lucia.createBlankSessionCookie().serialize());
    throw new TRPCError({ code: "UNAUTHORIZED", message: "No session found" });
  }
  const { session, user } = await lucia.validateSession(sessionId);
  if (!session || !user) {
    // resHeaders.append("Set-Cookie", lucia.createBlankSessionCookie().serialize());
    throw new TRPCError({ code: "UNAUTHORIZED", message: "Session not found" });
  }

  return {
    req,
    user: { id: user.id },
    tenant: { id: user.id },
  };
}

export type Context = inferAsyncReturnType<typeof createContext>;
