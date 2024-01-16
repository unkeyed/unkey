import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { unkeyRoleValidation } from "@unkey/rbac";
import { z } from "zod";
import { auth, t } from "../trpc";

export const permissionRouter = t.router({
  addRoleToRootKey: t.procedure
    .use(auth)
    .input(
      z.object({
        rootKeyId: z.string(),
        role: z.string(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const role = unkeyRoleValidation.safeParse(input.role);
      if (!role.success) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: `invalid role [${input.role}]: ${role.error.message}`,
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
      });
      if (!rootKey) {
        throw new TRPCError({ code: "NOT_FOUND", message: "root key not found" });
      }

      await db.insert(schema.roles).values({
        id: newId("role"),
        workspaceId: workspace.id,
        keyId: rootKey.id,
        role: role.data,
      });
    }),
  removeRoleFromRootKey: t.procedure
    .use(auth)
    .input(
      z.object({
        rootKeyId: z.string(),
        role: z.string(),
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
          message: "workspace not found",
        });
      }

      const role = await db.query.roles.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.role, input.role), eq(table.keyId, input.rootKeyId)),
        with: {
          key: true,
        },
      });

      if (!role || role.key.forWorkspaceId !== workspace.id) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "role not found",
        });
      }

      await db.delete(schema.roles).where(eq(schema.roles.id, role.id));
    }),
});
