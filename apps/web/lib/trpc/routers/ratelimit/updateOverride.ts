import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { auth, t } from "../../trpc";

export const updateOverride = t.procedure
  .use(auth)
  .input(
    z.object({
      id: z.string(),
      limit: z.number(),
      duration: z.number(),
      async: z.boolean().nullable(),
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

    await db
      .update(schema.ratelimitOverrides)
      .set({
        limit: input.limit,
        duration: input.duration,
        updatedAt: new Date(),
        async: input.async,
      })
      .where(eq(schema.ratelimitOverrides.id, override.id));
    await ingestAuditLogs({
      workspaceId: override.namespace.workspace.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "ratelimitOverride.update",
      description: `Changed ${override.id} limits from ${override.limit}/${override.duration} to ${input.limit}/${input.duration}`,
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
