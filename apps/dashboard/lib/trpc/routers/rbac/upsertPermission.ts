import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

const nameSchema = z
  .string()
  .min(3)
  .regex(/^[a-zA-Z0-9_:\-\.\*]+$/, {
    message:
      "Must be at least 3 characters long and only contain alphanumeric, colons, periods, dashes and underscores",
  });

export const upsertPermission = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      name: nameSchema,
    })
  )
  .mutation(async ({ input, ctx }) => {
    const workspaceId = ctx.workspace.id;
    const userId = ctx.user.id;

    return await db.transaction(async (tx) => {
      const existingPermission = await tx.query.permissions
        .findFirst({
          where: (table, { eq, and }) =>
            and(eq(table.name, input.name), eq(table.workspaceId, workspaceId)),
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
              "We are unable to upsert the permission. Please try again or contact support@unkey.dev",
          });
        });

      if (existingPermission) {
        return existingPermission;
      }

      const permission = {
        id: newId("permission"),
        workspaceId,
        name: input.name,
        slug: input.name,
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
              "We are unable to upsert the permission. Please try again or contact support@unkey.dev.",
          });
        });

      await insertAuditLogs(tx, {
        workspaceId,
        actor: { type: "user", id: userId },
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
            "We are unable to upsert the permission. Please try again or contact support@unkey.dev",
        });
      });

      return permission;
    });
  });
