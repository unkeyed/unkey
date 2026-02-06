import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const connectPermissionToRole = workspaceProcedure
  .input(
    z.object({
      roleId: z.string(),
      permissionId: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.orgId, ctx.tenant.id), isNull(table.deletedAtM)),
        with: {
          roles: {
            where: (table, { eq }) => eq(table.id, input.roleId),
          },
          permissions: {
            where: (table, { eq }) => eq(table.id, input.permissionId),
          },
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to connect this permission to role. Please try again or contact support@unkey.com",
        });
      });
    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct workspace. Please try again or contact support@unkey.com.",
      });
    }
    const role = workspace.roles.at(0);
    if (!role) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct role. Please try again or contact support@unkey.com.",
      });
    }
    const permission = workspace.permissions.at(0);
    if (!permission) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct permission. Please try again or contact support@unkey.com.",
      });
    }

    const tuple = {
      workspaceId: workspace.id,
      permissionId: permission.id,
      roleId: role.id,
    };
    await db
      .transaction(async (tx) => {
        await tx
          .insert(schema.rolesPermissions)
          .values({ ...tuple, createdAtM: Date.now() })
          .onDuplicateKeyUpdate({
            set: { ...tuple, updatedAtM: Date.now() },
          })
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to connect the permission to the role. Please try again or contact support@unkey.com.",
            });
          });
        await insertAuditLogs(tx, {
          workspaceId: workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "authorization.connect_role_and_permission",
          description: `Connect role ${role.id} to ${permission.id}`,
          resources: [
            {
              type: "role",
              id: role.id,
              name: role.name,
            },
            {
              type: "permission",
              id: permission.id,
              name: permission.name,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to connect this permission to role. Please try again or contact support@unkey.com",
        });
      });
  });
