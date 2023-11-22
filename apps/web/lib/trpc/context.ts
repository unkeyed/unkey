import { inferAsyncReturnType } from "@trpc/server";
import type { FetchCreateContextFnOptions } from "@trpc/server/adapters/fetch";

import { getAuth } from "@clerk/nextjs/server";
import { unkeyRoot } from "../api";
import { db, eq, schema } from "../db";

export async function createContext({ req, resHeaders }: FetchCreateContextFnOptions) {
  const { userId, orgId, orgRole } = getAuth(req as any);

  const tenantId = orgId ?? userId;
  const rootKey = tenantId ? await getRootKey(tenantId, req.headers, resHeaders) : undefined;

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

async function getRootKey(
  tenantId: string,
  reqHeaders: Headers,
  resHeaders: Headers,
): Promise<string | null> {
  const cookieName = `unkey_root_${tenantId}`;
  const rootKey = reqHeaders
    .get("Cookie")
    ?.split(" ")
    .find((c) => c.startsWith(cookieName))
    ?.split("=")
    .at(1)
    ?.replace(/;$/, "");
  if (rootKey) {
    return rootKey;
  }
  const ws = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!ws) {
    console.warn(`no workspace found for ${tenantId}`);
    return null;
  }
  const created = await unkeyRoot._internal.createRootKey({
    name: "tRPC",
    expires: Date.now() + 5 * 60 * 1000,
    forWorkspaceId: ws.id,
  });
  if (created.error) {
    console.error(created.error.message);
    return null;
  }
  created.result.key;

  let cookie = `${cookieName}=${created.result.key}; Max-Age=60; HttpOnly`;
  if (process.env.VERCEL) {
    cookie += "; secure; domain=unkey.dev";
  }
  resHeaders.set("Set-Cookie", cookie);

  return created.result.key ?? null;
}
