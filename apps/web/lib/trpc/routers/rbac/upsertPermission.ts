import { type Permission, db, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { newId } from "@unkey/id";
import { Context } from "../../context";

export async function upsertPermission(
  ctx: Context,
  workspaceId: string,
  name: string,
): Promise<Permission> {
  return await db.transaction(async (tx) => {
    const existingPermission = await tx.query.permissions.findFirst({
      where: (table, { and, eq }) => and(eq(table.workspaceId, workspaceId), eq(table.name, name)),
    });
    if (existingPermission) {
      return existingPermission;
    }

    const permission: Permission = {
      id: newId("permission"),
      workspaceId,
      name,
      description: null,
      createdAt: new Date(),
      updatedAt: null,
    };

    await tx.insert(schema.permissions).values(permission);
    await ingestAuditLogs({
      workspaceId,
      actor: { type: "user", id: ctx.user!.id },
      event: "permission.create",
      description: `Created ${permission.id}`,
      resources: [
        {
          type: "permission",
          id: permission.id,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });
    return permission;
  });
}
