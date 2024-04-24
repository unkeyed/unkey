import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { auth, t } from "../../trpc";

export const deleteOverride = t.procedure
  .use(auth)
  .input(
    z.object({
      id: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const override = await db.query.ratelimitOverrides.findFirst({
      where: (table, { and, eq, isNull }) => and(eq(table.id, input.id), isNull(table.deletedAt)),
      with: {
        namespace: {
          columns: {
            id: true,
          },
          with: {
            workspace: {
              columns: {
                id: true,
                tenantId: true,
              },
            },
          },
        },
      },
    });

    if (!override || override.namespace.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({
        message: "not found",
        code: "NOT_FOUND",
      });
    }

    await db.delete(schema.ratelimitOverrides).where(eq(schema.ratelimitOverrides.id, override.id));
    await ingestAuditLogs({
      workspaceId: override.namespace.workspace.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "ratelimitOverride.delete",
      description: `Deleted ${override.id}`,
      resources: [
        {
          type: "ratelimitNamespace",
          id: override.namespace.id,
        },
        {
          type: "ratelimitOverride",
          id: override.id,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });
  });
