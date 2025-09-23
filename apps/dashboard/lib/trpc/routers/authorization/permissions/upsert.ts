import { permissionSchema } from "@/app/(app)/[workspace]/authorization/permissions/components/upsert-permission/upsert-permission.schema";
import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { requireUser, requireWorkspace, t } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";

export const upsertPermission = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(permissionSchema)
  .mutation(async ({ input, ctx }) => {
    const isUpdate = Boolean(input.permissionId);
    let permissionId = input.permissionId;

    if (!isUpdate) {
      permissionId = newId("permission");
      if (!permissionId) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Failed to generate permission ID",
        });
      }
    }

    if (!permissionId) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Invalid permission ID",
      });
    }

    await db.transaction(async (tx) => {
      if (isUpdate && input.permissionId) {
        const updatePermissionId: string = input.permissionId;

        // Get existing permission
        const existingPermission = await tx.query.permissions.findFirst({
          where: (table, { and, eq }) =>
            and(eq(table.id, updatePermissionId), eq(table.workspaceId, ctx.workspace.id)),
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
                ne(table.id, updatePermissionId),
              ),
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
                ne(table.id, updatePermissionId),
              ),
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
      } else {
        // Create mode - check for both name and slug conflicts
        const [nameConflict, slugConflict] = await Promise.all([
          await tx.query.permissions.findFirst({
            where: (table, { and, eq }) =>
              and(eq(table.workspaceId, ctx.workspace.id), eq(table.name, input.name)),
          }),
          await tx.query.permissions.findFirst({
            where: (table, { and, eq }) =>
              and(eq(table.workspaceId, ctx.workspace.id), eq(table.slug, input.slug)),
          }),
        ]);

        if (nameConflict) {
          throw new TRPCError({
            code: "CONFLICT",
            message: `Permission with name '${input.name}' already exists`,
          });
        }

        if (slugConflict) {
          throw new TRPCError({
            code: "CONFLICT",
            message: `Permission with slug '${input.slug}' already exists`,
          });
        }

        // Create new permission
        await tx
          .insert(schema.permissions)
          .values({
            id: permissionId,
            name: input.name,
            slug: input.slug,
            description: input.description,
            workspaceId: ctx.workspace.id,
          })
          .catch(() => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message: "Failed to create permission",
            });
          });

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          event: "permission.create",
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          description: `Created permission ${permissionId}`,
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
      }
    });

    return {
      permissionId,
      isUpdate,
      message: isUpdate ? "Permission updated successfully" : "Permission created successfully",
    };
  });
