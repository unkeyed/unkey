import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, inArray, schema } from "@/lib/db";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const deleteRoleWithRelations = workspaceProcedure
  .input(
    z.object({
      roleIds: z
        .union([z.string(), z.array(z.string())])
        .transform((ids) => (Array.isArray(ids) ? ids : [ids])),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    if (input.roleIds.length === 0) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "At least one role ID must be provided.",
      });
    }

    await db.transaction(async (tx) => {
      // Fetch all roles to validate existence and get names for audit logs
      const roles = await tx.query.roles.findMany({
        where: (table, { and, eq, inArray }) =>
          and(eq(table.workspaceId, ctx.workspace.id), inArray(table.id, input.roleIds)),
      });

      if (roles.length !== input.roleIds.length) {
        const foundIds = roles.map((r) => r.id);
        const missingIds = input.roleIds.filter((id) => !foundIds.includes(id));
        throw new TRPCError({
          code: "NOT_FOUND",
          message: `Role(s) not found: ${missingIds.join(
            ", ",
          )}. Please try again or contact support@unkey.com.`,
        });
      }

      // Delete related records first to avoid foreign key constraints
      await tx
        .delete(schema.rolesPermissions)
        .where(
          and(
            inArray(schema.rolesPermissions.roleId, input.roleIds),
            eq(schema.rolesPermissions.workspaceId, ctx.workspace.id),
          ),
        )
        .catch((err) => {
          console.error("Failed to delete role permissions:", err);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "We are unable to delete the roles. Please try again or contact support@unkey.com",
          });
        });

      await tx
        .delete(schema.keysRoles)
        .where(
          and(
            inArray(schema.keysRoles.roleId, input.roleIds),
            eq(schema.keysRoles.workspaceId, ctx.workspace.id),
          ),
        )
        .catch((err) => {
          console.error("Failed to delete key-role relationships:", err);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "We are unable to delete the roles. Please try again or contact support@unkey.com",
          });
        });

      // Delete the roles themselves
      await tx
        .delete(schema.roles)
        .where(
          and(
            inArray(schema.roles.id, input.roleIds),
            eq(schema.roles.workspaceId, ctx.workspace.id),
          ),
        )
        .catch((err) => {
          console.error("Failed to delete roles:", err);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "We are unable to delete the roles. Please try again or contact support@unkey.com",
          });
        });

      // Create single audit log for bulk delete
      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "role.delete",
        description: `Deleted ${roles.length} role(s): ${roles.map((r) => r.name).join(", ")}`,
        resources: roles.map((role) => ({
          type: "role",
          id: role.id,
          name: role.name,
        })),
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      }).catch((err) => {
        console.error("Failed to create audit log:", err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to delete the roles. Please try again or contact support@unkey.com.",
        });
      });
    });

    return { deletedCount: input.roleIds.length };
  });
