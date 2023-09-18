import { TRPCError, inferAsyncReturnType } from "@trpc/server";
import type { FetchCreateContextFnOptions } from "@trpc/server/adapters/fetch";

import { getAuth } from "@clerk/nextjs/server";
import { unkeyRoot } from "../api";
import { db, eq, schema } from "../db";

export async function createContext({ req, resHeaders }: FetchCreateContextFnOptions) {
  // rome-ignore lint/suspicious/noExplicitAny: TODO
  const { userId, orgId, orgRole } = getAuth(req as any);

  let rootKey: string | undefined;
  const tenantId = orgId ?? userId;
  if (tenantId) {
    const cookieName = `unkey_root_${tenantId}`;
    rootKey = req.headers
      .get("Cookie")
      ?.split(" ")
      .find((c) => c.startsWith(cookieName))
      ?.split("=")
      .at(1)
      ?.replace(/;$/, "");
    if (!rootKey) {
      const ws = await db.query.workspaces.findFirst({
        where: eq(schema.workspaces.tenantId, tenantId),
      });
      if (!ws) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: `no workspace found for ${tenantId}`,
        });
      }
      const created = await unkeyRoot._internal.createRootKey({
        name: "tRPC",
        expires: Date.now() + 60_000,
        forWorkspaceId: ws.id,
      });
      if (created.error) {
        throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: created.error.message });
      }
      rootKey = created.result.key;

      let cookie = `${cookieName}=${rootKey}; Max-Age=60; HttpOnly`;
      if (process.env.VERCEL) {
        cookie += "; secure; domain=unkey.dev";
      }
      resHeaders.set("Set-Cookie", cookie);
    }
  }
  if (!rootKey) {
    throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: "unable to find a root key" });
  }

  return {
    req,
    user: userId ? { id: userId } : null,
    rootKey,
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
