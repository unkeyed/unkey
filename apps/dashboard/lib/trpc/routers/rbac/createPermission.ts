import { db, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { auth, t } from "../../trpc";

const nameSchema = z
  .string()
  .min(3)
  .regex(/^[a-zA-Z0-9_:\-\.\*]+$/, {
    message:
      "Must be at least 3 characters long and only contain alphanumeric, colons, periods, dashes and underscores",
  });

export const createPermission = t.procedure
  .use(auth)
  .input(
    z.object({
      name: nameSchema,
      description: z.string().optional(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
    });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "workspace not found",
      });
    }
    const permissionId = newId("permission");
    await db.insert(schema.permissions).values({
      id: permissionId,
      name: input.name,
      description: input.description,
      workspaceId: workspace.id,
    });
    await ingestAuditLogs({
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

    return { permissionId };
  });
