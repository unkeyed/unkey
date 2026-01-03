import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
const nameSchema = z
  .string()
  .min(3)
  .regex(/^[a-zA-Z0-9_:\-\.\*]+$/, {
    message:
      "Must be at least 3 characters long and only contain alphanumeric, colons, periods, dashes and underscores",
  });

export const createPermission = workspaceProcedure
  .input(
    z.object({
      name: nameSchema,
      description: z.string().optional(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const permissionId = newId("permission");
    await db
      .transaction(async (tx) => {
        const existing = await tx.query.permissions.findFirst({
          where: (table, { and, eq }) =>
            and(eq(table.workspaceId, ctx.workspace.id), eq(table.name, input.name)),
        });
        if (existing) {
          throw new TRPCError({
            code: "CONFLICT",
            message:
              "Permission with the same name already exists. To update it, go to 'Authorization' in the sidebar.",
          });
        }

        await tx.insert(schema.permissions).values({
          id: permissionId,
          name: input.name,
          slug: input.name,
          description: input.description,
          workspaceId: ctx.workspace.id,
        });
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          event: "permission.create",
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          description: `Created ${permissionId}`,
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
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to create a permission. Please try again or contact support@unkey.dev.",
        });
      });

    return { permissionId };
  });
