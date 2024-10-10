import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";

const nameSchema = z
  .string()
  .min(3)
  .regex(/^[a-zA-Z0-9_:\-\.\*]+$/, {
    message:
      "Must be at least 3 characters long and only contain alphanumeric, colons, periods, dashes and underscores",
  });

export const createPermission = rateLimitedProcedure(ratelimit.create)
  .input(
    z.object({
      name: nameSchema,
      description: z.string().optional(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to create permission. Please try again or contact support@unkey.dev",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct workspace. Please try again or contact support@unkey.dev.",
      });
    }
    const permissionId = newId("permission");
    await db
      .transaction(async (tx) => {
        await tx.insert(schema.permissions).values({
          id: permissionId,
          name: input.name,
          description: input.description,
          workspaceId: workspace.id,
        });
        await insertAuditLogs(tx, {
          workspaceId: workspace.id,
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
