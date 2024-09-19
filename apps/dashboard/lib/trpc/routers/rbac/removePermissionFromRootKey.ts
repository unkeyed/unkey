import { and, db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const removePermissionFromRootKey = rateLimitedProcedure(ratelimit.update)
  .input(
    z.object({
      rootKeyId: z.string(),
      permissionName: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
    });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct workspace. Please contact support using support@unkey.dev.",
      });
    }

    const key = await db.query.keys.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(
          eq(schema.keys.forWorkspaceId, workspace.id),
          eq(schema.keys.id, input.rootKeyId),
          isNull(table.deletedAt),
        ),
      with: {
        permissions: {
          with: {
            permission: true,
          },
        },
      },
    });
    if (!key) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Key ${input.rootKeyId} not found`,
      });
    }

    const permissionRelation = key.permissions.find(
      (kp) => kp.permission.name === input.permissionName,
    );
    if (!permissionRelation) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Key ${input.rootKeyId} did not have permission ${input.permissionName}`,
      });
    }

    await db
      .delete(schema.keysPermissions)
      .where(
        and(
          eq(schema.keysPermissions.keyId, permissionRelation.keyId),
          eq(schema.keysPermissions.workspaceId, permissionRelation.workspaceId),
          eq(schema.keysPermissions.permissionId, permissionRelation.permissionId),
        ),
      )
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to remove the permission from the key. Please contact support using support@unkey.dev",
        });
      });

    await ingestAuditLogs({
      workspaceId: workspace.id,
      actor: { type: "user", id: ctx.user.id },
      event: "authorization.disconnect_permission_and_key",
      description: `Disconnect ${input.permissionName} from ${input.rootKeyId}`,
      resources: [
        {
          type: "permission",
          id: input.permissionName,
        },
        {
          type: "key",
          id: input.rootKeyId,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });
  });
