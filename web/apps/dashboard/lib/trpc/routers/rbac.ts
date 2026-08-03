import { type InsertPermission, type Permission, db, schema } from "@/lib/db";

import type { UnkeyAuditLog } from "@/lib/audit";
import { ensureDefaultProjectId } from "@/lib/projects/ensure-default-project-id";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import type { Context } from "../context";

export async function upsertPermissions(
  ctx: Context,
  workspaceId: string,
  slugs: string[],
): Promise<{
  permissions: Omit<Permission, "pk">[];
  auditLogs: UnkeyAuditLog[];
}> {
  return await db.transaction(async (tx) => {
    const projectId = await ensureDefaultProjectId(tx, workspaceId);
    const existingPermissions = await tx.query.permissions.findMany({
      where: (table, { inArray, and, eq }) =>
        and(eq(table.workspaceId, workspaceId), inArray(table.slug, slugs)),
      columns: {
        id: true,
        workspaceId: true,
        projectId: true,
        name: true,
        slug: true,
        description: true,
        updatedAtM: true,
        createdAtM: true,
      },
    });

    const newPermissions: InsertPermission[] = [];
    const auditLogs: UnkeyAuditLog[] = [];

    const userId = ctx.user?.id;
    if (!userId) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "UserId not found",
      });
    }

    const permissions: Omit<Permission, "pk">[] = slugs.map((slug) => {
      const existingPermission = existingPermissions.find((p) => p.slug === slug);

      if (existingPermission) {
        return existingPermission;
      }

      const permission = {
        id: newId("permission"),
        workspaceId,
        projectId,
        name: slug,
        slug: slug,
        description: null,
        updatedAtM: null,
        createdAtM: Date.now(),
      };

      newPermissions.push(permission);
      auditLogs.push({
        workspaceId,
        actor: { type: "user", id: userId },
        event: "permission.create",
        description: `Created ${permission.id}`,
        resources: [
          {
            type: "permission",
            id: permission.id,
            name: permission.slug,
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });

      return permission;
    });

    if (newPermissions.length) {
      await tx.insert(schema.permissions).values(newPermissions);
    }

    return { permissions, auditLogs };
  });
}
