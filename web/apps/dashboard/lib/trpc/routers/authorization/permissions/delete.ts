import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, inArray, schema } from "@/lib/db";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const deletePermissionWithRelations = workspaceProcedure
  .input(
    z.object({
      permissionIds: z
        .union([z.string(), z.array(z.string())])
        .transform((ids) => (Array.isArray(ids) ? ids : [ids])),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    if (input.permissionIds.length === 0) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "At least one permission ID must be provided.",
      });
    }

    await db.transaction(async (tx) => {
      // Fetch all permissions to validate existence and get names for audit logs
      const permissions = await tx.query.permissions.findMany({
        where: (table, { and, eq, inArray }) =>
          and(eq(table.workspaceId, ctx.workspace.id), inArray(table.id, input.permissionIds)),
      });

      if (permissions.length !== input.permissionIds.length) {
        const foundIds = permissions.map((p) => p.id);
        const missingIds = input.permissionIds.filter((id) => !foundIds.includes(id));
        throw new TRPCError({
          code: "NOT_FOUND",
          message: `Permission(s) not found: ${missingIds.join(
            ", ",
          )}. Please try again or contact support@unkey.dev.`,
        });
      }

      // Delete related records first to avoid foreign key constraints
      // Delete roles_permissions relationships
      await tx
        .delete(schema.rolesPermissions)
        .where(
          and(
            inArray(schema.rolesPermissions.permissionId, input.permissionIds),
            eq(schema.rolesPermissions.workspaceId, ctx.workspace.id),
          ),
        )
        .catch((err) => {
          console.error("Failed to delete role-permission relationships:", err);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "We are unable to delete the permissions. Please try again or contact support@unkey.dev",
          });
        });

      // Delete keys_permissions relationships
      await tx
        .delete(schema.keysPermissions)
        .where(
          and(
            inArray(schema.keysPermissions.permissionId, input.permissionIds),
            eq(schema.keysPermissions.workspaceId, ctx.workspace.id),
          ),
        )
        .catch((err) => {
          console.error("Failed to delete key-permission relationships:", err);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "We are unable to delete the permissions. Please try again or contact support@unkey.dev",
          });
        });

      // Delete the permissions themselves
      await tx
        .delete(schema.permissions)
        .where(
          and(
            inArray(schema.permissions.id, input.permissionIds),
            eq(schema.permissions.workspaceId, ctx.workspace.id),
          ),
        )
        .catch((err) => {
          console.error("Failed to delete permissions:", err);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "We are unable to delete the permissions. Please try again or contact support@unkey.dev",
          });
        });

      // Create single audit log for bulk delete
      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "permission.delete",
        description: `Deleted ${permissions.length} permission(s): ${permissions
          .map((p) => p.name)
          .join(", ")}`,
        resources: permissions.map((permission) => ({
          type: "permission",
          id: permission.id,
          name: permission.name,
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
            "We are unable to delete the permissions. Please try again or contact support@unkey.dev.",
        });
      });
    });

    return { deletedCount: input.permissionIds.length };
  });
