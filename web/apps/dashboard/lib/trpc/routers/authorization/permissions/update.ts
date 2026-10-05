import { permissionSchema } from "@/app/(app)/[workspaceSlug]/authorization/permissions/components/upsert-permission/upsert-permission.schema";
import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const updatePermission = workspaceProcedure
  .input(permissionSchema.extend({ permissionId: z.string().startsWith("perm_") }))
  .mutation(async ({ input, ctx }) => {
    const permissionId = input.permissionId;

    await db.transaction(async (tx) => {
      // Get existing permission
      const existingPermission = await tx.query.permissions.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.id, permissionId), eq(table.workspaceId, ctx.workspace.id)),
        columns: {
          id: true,
          name: true,
          slug: true,
        },
      });

      if (!existingPermission) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Permission not found or access denied",
        });
      }

      // Check for name conflicts only if name is changing
      if (existingPermission.name !== input.name) {
        const nameConflict = await tx.query.permissions.findFirst({
          where: (table, { and, eq, ne }) =>
            and(
              eq(table.workspaceId, ctx.workspace.id),
              eq(table.name, input.name),
              ne(table.id, permissionId),
            ),
          columns: { id: true },
        });

        if (nameConflict) {
          throw new TRPCError({
            code: "CONFLICT",
            message: `Permission with name '${input.name}' already exists`,
          });
        }
      }

      // Check for slug conflicts only if slug is changing
      if (existingPermission.slug !== input.slug) {
        const slugConflict = await tx.query.permissions.findFirst({
          where: (table, { and, eq, ne }) =>
            and(
              eq(table.workspaceId, ctx.workspace.id),
              eq(table.slug, input.slug),
              ne(table.id, permissionId),
            ),
          columns: { id: true },
        });

        if (slugConflict) {
          throw new TRPCError({
            code: "CONFLICT",
            message: `Permission with slug '${input.slug}' already exists`,
          });
        }
      }

      // Update permission
      await tx
        .update(schema.permissions)
        .set({
          name: input.name,
          slug: input.slug,
          description: input.description,
          updatedAtM: Date.now(),
        })
        .where(
          and(
            eq(schema.permissions.id, permissionId),
            eq(schema.permissions.workspaceId, ctx.workspace.id),
          ),
        )
        .catch(() => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Failed to update permission",
          });
        });

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        event: "permission.update",
        actor: {
          type: "user",
          id: ctx.user.id,
        },
        description: `Updated permission ${permissionId}`,
        resources: [
          {
            type: "permission",
            id: permissionId,
            name: input.name,
          },
        ],
        context: {
          userAgent: ctx.audit.userAgent,
          location: ctx.audit.location,
        },
      });
    });

    return { permissionId, message: "Permission updated successfully" };
  });
