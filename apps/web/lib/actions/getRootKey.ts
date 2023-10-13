"use server";

import { auth } from "@clerk/nextjs";
import { Result, result } from "@unkey/result";
import { cookies } from "next/headers";
import { unkeyRoot } from "../api";
import { db, eq, schema } from "../db";

/**
 * getRootKey loads the root key from a cookie or creates a new one if required
 */
export async function getRootKey(): Promise<Result<string>> {
  const { userId, orgId } = auth();

  const tenantId = orgId ?? userId;
  if (!tenantId) {
    return result.fail({ message: "unable to get tenantId" });
  }

  const cookieName = `unkey_root_${tenantId}`;
  const cookie = cookies().get(cookieName);
  if (cookie) {
    return result.success(cookie.value);
  }
  const ws = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!ws) {
    return result.fail({ message: `no workspace found for ${tenantId}` });
  }

  const created = await unkeyRoot._internal.createRootKey({
    name: "Dashboard",
    expires: Date.now() + 5 * 60 * 1000, // 5min
    forWorkspaceId: ws.id,
  });
  if (created.error) {
    console.error(created.error.message);
    return result.fail(created.error);
  }
  cookies().set(cookieName, created.result.key, {
    maxAge: 5 * 60,
    httpOnly: true,
    secure: !!process.env.VERCEL,
    domain: process.env.VERCEL ? "unkey.dev" : undefined,
  });

  return result.success(created.result.key);
}
