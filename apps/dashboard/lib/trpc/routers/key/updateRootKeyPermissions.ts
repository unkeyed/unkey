import type { UnkeyAuditLog } from "@/lib/audit";
import { and, db, eq, inArray, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { unkeyPermissionValidation } from "@unkey/rbac";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

import { insertAuditLogs } from "@/lib/audit";
import { env } from "@/lib/env";
import { upsertPermissions } from "../rbac";

/**
 * Replaces the full permission set for the root key — clients must submit the complete,
 * authoritative permission list to avoid lost updates
 */
export const updateRootKeyPermissions = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      keyId: z.string(),
      permissions: z.array(unkeyPermissionValidation).min(1, {
        message: "You need to add at least one permission.",
      }),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    // Verify the key exists and belongs to the workspace
    const key = await db.query.keys
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(
            eq(table.id, input.keyId),
            eq(table.forWorkspaceId, ctx.workspace.id),
            isNull(table.deletedAtM),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to update the root key permissions. Please try again or contact support@unkey.dev.",
        });
      });

    if (!key) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Root key not found",
      });
    }

    const auditLogs: UnkeyAuditLog[] = [];

    try {
      await db.transaction(async (tx) => {
        // Get current permissions for audit logging
        const currentPermissions = await tx.query.keysPermissions
          .findMany({
            where: (table, { eq }) => eq(table.keyId, input.keyId),
            with: {
              permission: true,
            },
          })
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message: "Failed to retrieve current permissions",
            });
          });

        // Upsert new permissions
        const { permissions: upsertedPermissions, auditLogs: createPermissionLogs } =
          await upsertPermissions(ctx, env().UNKEY_WORKSPACE_ID, input.permissions);

        auditLogs.push(...createPermissionLogs);

        // Get current permission IDs for comparison
        const currentPermissionIds = new Set(currentPermissions.map((kp) => kp.permissionId));
        const newPermissionIds = new Set(upsertedPermissions.map((p) => p.id));

        // Find permissions to remove (in current but not in new)
        const permissionsToRemove = currentPermissions.filter(
          (kp) => !newPermissionIds.has(kp.permissionId),
        );

        // Find permissions to add (in new but not in current)
        const permissionsToAdd = upsertedPermissions.filter((p) => !currentPermissionIds.has(p.id));

        // Remove only the permissions that are no longer needed
        if (permissionsToRemove.length > 0) {
          const permissionIdsToRemove = permissionsToRemove.map((kp) => kp.permissionId);
          await tx
            .delete(schema.keysPermissions)
            .where(
              and(
                eq(schema.keysPermissions.keyId, input.keyId),
                inArray(schema.keysPermissions.permissionId, permissionIdsToRemove),
              ),
            )
            .catch((_err) => {
              throw new TRPCError({
                code: "INTERNAL_SERVER_ERROR",
                message: "Failed to remove existing permissions",
              });
            });

          // Audit log for removed permissions
          auditLogs.push(
            ...permissionsToRemove.map((kp) => ({
              workspaceId: ctx.workspace.id,
              actor: { type: "user" as const, id: ctx.user.id },
              event: "authorization.disconnect_permission_and_key" as const,
              description: `Disconnected ${kp.permissionId} from ${input.keyId}`,
              resources: [
                {
                  type: "key" as const,
                  id: input.keyId,
                  name: key.name ?? undefined,
                },
                {
                  type: "permission" as const,
                  id: kp.permissionId,
                  name: kp.permission?.name ?? undefined,
                },
              ],
              context: {
                location: ctx.audit.location,
                userAgent: ctx.audit.userAgent,
              },
            })),
          );
        }

        // Add only the new permissions
        if (permissionsToAdd.length > 0) {
          await tx.insert(schema.keysPermissions).values(
            permissionsToAdd.map((p) => ({
              keyId: input.keyId,
              permissionId: p.id,
              workspaceId: env().UNKEY_WORKSPACE_ID,
            })),
          );

          // Audit log for new permission connections
          auditLogs.push(
            ...permissionsToAdd.map((p) => ({
              workspaceId: ctx.workspace.id,
              actor: { type: "user" as const, id: ctx.user.id },
              event: "authorization.connect_permission_and_key" as const,
              description: `Connected ${p.id} and ${input.keyId}`,
              resources: [
                {
                  type: "key" as const,
                  id: input.keyId,
                  name: key.name ?? undefined,
                },
                {
                  type: "permission" as const,
                  id: p.id,
                  name: p.name ?? undefined,
                },
              ],
              context: {
                location: ctx.audit.location,
                userAgent: ctx.audit.userAgent,
              },
            })),
          );
        }

        await insertAuditLogs(tx, auditLogs);
      });
    } catch (_err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "We are unable to update the root key permissions. Please try again or contact support@unkey.dev",
      });
    }

    return { success: true };
  });
