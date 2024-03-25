import { db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { unkeyPermissionValidation } from "@unkey/rbac";
import { z } from "zod";
import { auth, t } from "../../trpc";
import { upsertPermission } from "../rbac";

export const addPermissionToRootKey = t.procedure
  .use(auth)
  .input(
    z.object({
      rootKeyId: z.string(),
      permission: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const permission = unkeyPermissionValidation.safeParse(input.permission);
    if (!permission.success) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: `invalid permission [${input.permission}]: ${permission.error.message}`,
      });
    }

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

    const rootKey = await db.query.keys.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.forWorkspaceId, workspace.id), eq(table.id, input.rootKeyId)),
      with: {
        permissions: {
          with: {
            permission: true,
          },
        },
      },
    });
    if (!rootKey) {
      throw new TRPCError({ code: "NOT_FOUND", message: "root key not found" });
    }

    const p = await upsertPermission(ctx, rootKey.workspaceId, permission.data);

    await db
      .insert(schema.keysPermissions)
      .values({
        keyId: rootKey.id,
        permissionId: p.id,
        workspaceId: p.workspaceId,
      })
      .onDuplicateKeyUpdate({ set: { permissionId: p.id } });
  });
