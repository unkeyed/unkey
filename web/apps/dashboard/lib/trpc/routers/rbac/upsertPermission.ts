import { insertAuditLogs } from "@/lib/audit";
import { type Permission, db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import type { Context } from "../../context";

export async function upsertPermission(
  ctx: Context,
  workspaceId: string,
  name: string,
): Promise<Omit<Permission, "pk">> {
  return await db.transaction(async (tx) => {
    const existingPermission = await tx.query.permissions
      .findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.name, name), eq(table.workspaceId, workspaceId)),
        with: {
          workspace: {
            columns: { id: true },
          },
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to upsert the permission. Please try again or contact support@unkey.com",
        });
      });

    if (existingPermission) {
      return existingPermission;
    }

    const permission = {
      id: newId("permission"),
      workspaceId,
      name,
      slug: name,
      description: null,
      createdAtM: Date.now(),
      updatedAtM: null,
    };

    await tx
      .insert(schema.permissions)
      .values(permission)
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to upsert the permission. Please try again or contact support@unkey.com.",
        });
      });
    await insertAuditLogs(tx, {
      workspaceId,
      // biome-ignore lint/style/noNonNullAssertion: Safe to leave
      actor: { type: "user", id: ctx.user!.id },
      event: "permission.create",
      description: `Created ${permission.id}`,
      resources: [
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
    }).catch((_err) => {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "We are unable to upsert the permission. Please try again or contact support@unkey.com",
      });
    });

    return permission;
  });
}
