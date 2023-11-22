import { auth } from "@clerk/nextjs";
import { z } from "zod";

import { unkeyRoot } from "@/lib/api";
import { db, eq, schema } from "@/lib/db";
import { Result, result } from "@unkey/result";
import { cookies } from "next/headers";

export function serverAction<TInput, TOutput = void>(opts: {
  input: z.ZodSchema<TInput, any, any>;
  output?: z.ZodSchema<TOutput>;
  handler: (args: {
    input: TInput;
    ctx: { tenantId: string; userId: string; rootKey: string };
  }) => Promise<TOutput>;
}): (formData: FormData) => Promise<Result<TOutput>> {
  const { userId, orgId } = auth();
  const tenantId = orgId ?? userId;
  if (!tenantId) {
    throw new Error("unauthorized");
  }

  return async (formData: FormData) => {
    const req: Record<string, unknown> = {};
    formData.forEach((v, k) => {
      req[k] = v;
    });

    const input = opts.input.safeParse(req);
    if (!input.success) {
      return result.fail(input.error);
    }

    const rootKey = await getRootKey(tenantId);
    if (rootKey.error) {
      return result.fail(rootKey.error);
    }

    try {
      const res = await opts.handler({
        input: input.data,
        ctx: { tenantId, userId: userId!, rootKey: rootKey.value },
      });
      return result.success(res);
    } catch (e) {
      return result.fail({ message: (e as Error).message });
    }
  };
}

/**
 * getRootKey loads the root key from a cookie or creates a new one if required
 */
export async function getRootKey(tenantId: string): Promise<Result<string>> {
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
