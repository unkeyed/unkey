import { type Permission, db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { type UnkeyAuditLog, ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { newKey } from "@unkey/keys";
import { unkeyPermissionValidation } from "@unkey/rbac";
import { z } from "zod";
import { auth, t } from "../../trpc";
import { upsertPermissions } from "../rbac";

export const createRootKey = t.procedure
  .use(auth)
  .input(
    z.object({
      name: z.string().optional(),
      permissions: z.array(unkeyPermissionValidation).min(1, {
        message: "You must add at least 1 permissions",
      }),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
    });
    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "workspace not found",
      });
    }

    const unkeyApi = await db.query.apis.findFirst({
      where: eq(schema.apis.id, env().UNKEY_API_ID),
      with: {
        workspace: true,
      },
    });
    if (!unkeyApi) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `api ${env().UNKEY_API_ID} not found`,
      });
    }
    if (!unkeyApi.keyAuthId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: `api ${env().UNKEY_API_ID} is not setup to handle keys`,
      });
    }

    const keyId = newId("key");

    const { key, hash, start } = await newKey({
      prefix: "unkey",
      byteLength: 16,
    });

    const auditLogs: UnkeyAuditLog[] = [];

    await db.transaction(async (tx) => {
      await tx.insert(schema.keys).values({
        id: keyId,
        keyAuthId: unkeyApi.keyAuthId!,
        name: input?.name,
        hash,
        start,
        ownerId: ctx.user.id,
        workspaceId: env().UNKEY_WORKSPACE_ID,
        forWorkspaceId: workspace.id,
        expires: null,
        createdAt: new Date(),
        remaining: null,
        refillInterval: null,
        refillAmount: null,
        lastRefillAt: null,
        deletedAt: null,
        enabled: true,
      });

      auditLogs.push({
        workspaceId: workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "key.create",
        description: `Created ${keyId}`,
        resources: [
          {
            type: "key",
            id: keyId,
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });

      const { permissions, auditLogs: createPermissionLogs } = await upsertPermissions(
        ctx,
        env().UNKEY_WORKSPACE_ID,
        input.permissions,
      );
      auditLogs.push(...createPermissionLogs);

      auditLogs.push(
        ...permissions.map((p) => ({
          workspaceId: workspace.id,
          actor: { type: "user" as const, id: ctx.user.id },
          event: "authorization.connect_permission_and_key" as const,
          description: `Connected ${p.id} and ${keyId}`,
          resources: [
            {
              type: "key" as const,
              id: keyId,
            },
            {
              type: "permission" as const,
              id: p.id,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        })),
      );

      await tx.insert(schema.keysPermissions).values(
        permissions.map((p) => ({
          keyId,
          permissionId: p.id,
          workspaceId: env().UNKEY_WORKSPACE_ID,
        })),
      );
    });
    await ingestAuditLogs(auditLogs);

    return { key, keyId };
  });
