import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { unkeyPermissionValidation } from "@unkey/rbac";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";
import { upsertPermissions } from "../rbac";
export const addPermissionToRootKey = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      rootKeyId: z.string(),
      permission: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const permission = unkeyPermissionValidation.safeParse(input.permission);
    if (!permission.success) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: `Invalid permission [${input.permission}]: ${permission.error.message}`,
      });
    }

    const rootKey = await db.query.keys.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.forWorkspaceId, ctx.workspace.id), eq(table.id, input.rootKeyId)),
      with: {
        permissions: {
          with: {
            permission: true,
          },
        },
      },
    });
    if (!rootKey) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct root key. Please try again or contact support@unkey.dev.",
      });
    }

    const { permissions, auditLogs } = await upsertPermissions(ctx, rootKey.workspaceId, [
      permission.data,
    ]);
    const p = permissions[0];
    await db
      .transaction(async (tx) => {
        await tx
          .insert(schema.keysPermissions)
          .values({
            keyId: rootKey.id,
            permissionId: p.id,
            workspaceId: p.workspaceId,
          })
          .onDuplicateKeyUpdate({ set: { permissionId: p.id } })
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to add permission to the root key. Please try again or contact support@unkey.dev.",
            });
          });
        await insertAuditLogs(tx, [
          ...auditLogs,
          {
            workspaceId: ctx.workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "authorization.connect_permission_and_key",
            description: `Attached ${p.id} to ${rootKey.id}`,
            resources: [
              {
                type: "key",
                id: rootKey.id,
              },
              {
                type: "permission",
                id: p.id,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          },
        ]);
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to add permission to the rootkey. Please try again or contact support@unkey.dev",
        });
      });
  });
