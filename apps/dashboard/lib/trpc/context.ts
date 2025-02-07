import type { inferAsyncReturnType } from "@trpc/server";
import type { FetchCreateContextFnOptions } from "@trpc/server/adapters/fetch";

import { getAuth } from "@/lib/auth/get-auth";
import { newId } from "@unkey/id";
import { type AuditLogBucket, type Workspace, db, schema } from "../db";

export async function createContext({ req }: FetchCreateContextFnOptions) {
  const { userId, orgId, orgRole } = await getAuth(req as any);

  let ws: (Workspace & { auditLogBucket: AuditLogBucket }) | undefined;
  const tenantId = orgId ?? userId;
  if (tenantId) {
    await db.transaction(async (tx) => {
      const res = await tx.query.workspaces.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
        with: {
          auditLogBuckets: {
            where: (table, { eq }) => eq(table.name, "unkey_mutations"),
          },
        },
      });
      if (res) {
        let auditLogBucket = res.auditLogBuckets.at(0);
        // @ts-expect-error it should be undefined
        delete res.auditLogBuckets; // we don't need to pollute or context
        if (!auditLogBucket) {
          auditLogBucket = {
            id: newId("auditLogBucket"),
            name: "unkey_mutations",
            createdAt: Date.now(),
            deleteProtection: true,
            workspaceId: res.id,
            retentionDays: 30,
            updatedAt: null,
          };
          await tx.insert(schema.auditLogBucket).values(auditLogBucket);
        }
        ws = {
          ...res,
          auditLogBucket,
        };
      }
    });
  }

  return {
    req,
    audit: {
      userAgent: req.headers.get("user-agent") ?? undefined,
      location: req.headers.get("x-forwarded-for") ?? process.env.VERCEL_REGION ?? "unknown",
    },
    user: userId ? { id: userId } : null,
    workspace: ws,
    tenant:
      orgId && orgRole
        ? {
            id: orgId,
            role: orgRole,
          } : null
 }
}   

export type Context = inferAsyncReturnType<typeof createContext>;
