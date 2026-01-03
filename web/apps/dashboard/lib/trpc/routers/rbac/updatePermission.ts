import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
const nameSchema = z
  .string()
  .min(3)
  .regex(/^[a-zA-Z0-9_:\-\.\*]+$/, {
    message:
      "Must be at least 3 characters long and only contain alphanumeric, colons, periods, dashes and underscores",
  });

export const updatePermission = workspaceProcedure
  .input(
    z.object({
      id: z.string(),
      name: nameSchema,
      description: z.string().nullable(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const permission = await db.query.permissions.findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.workspaceId, ctx.workspace.id), eq(table.id, input.id)),
    });

    if (!permission) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct permission. Please try again or contact support@unkey.dev.",
      });
    }

    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.permissions)
          .set({
            name: input.name,
            description: input.description,
            updatedAtM: Date.now(),
          })
          .where(eq(schema.permissions.id, permission.id));
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "permission.update",
          description: `Update permission ${input.id}`,
          resources: [
            {
              type: "permission",
              id: input.id,
              name: input.name,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      })

      .catch((err) => {
        console.error(err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update the permission. Please try again or contact support@unkey.dev.",
        });
      });
  });
