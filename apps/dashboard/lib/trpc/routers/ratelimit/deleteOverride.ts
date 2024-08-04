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
        message:
          "We are unable to find the correct override. Please contact support using support@unkey.dev.",
        code: "NOT_FOUND",
      });
    }

    await db
      .update(schema.ratelimitOverrides)
      .set({ deletedAt: new Date() })
      .where(eq(schema.ratelimitOverrides.id, override.id))
      .catch((_err) => {
        throw new TRPCError({
          message:
            "We are unable to delete the override. Please contact support using support@unkey.dev",
          code: "INTERNAL_SERVER_ERROR",
        });
      });

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
