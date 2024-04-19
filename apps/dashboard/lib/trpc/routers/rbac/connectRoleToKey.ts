import { db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const connectRoleToKey = t.procedure
  .use(auth)
  .input(
    z.object({
      roleId: z.string(),
      keyId: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      with: {
        roles: {
          where: (table, { eq }) => eq(table.id, input.roleId),
        },
        keys: {
          where: (table, { eq }) => eq(table.id, input.keyId),
        },
      },
    });
    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "workspace not found",
      });
    }
    const role = workspace.roles.at(0);
    if (!role) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "role not found",
      });
    }
    const key = workspace.keys.at(0);
    if (!key) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "key not found",
      });
    }

    const tuple = {
      workspaceId: workspace.id,
      keyId: key.id,
      roleId: role.id,
    };
    await db
      .insert(schema.keysRoles)
      .values({ ...tuple, createdAt: new Date() })
      .onDuplicateKeyUpdate({
        set: { ...tuple, updatedAt: new Date() },
      });
  });
