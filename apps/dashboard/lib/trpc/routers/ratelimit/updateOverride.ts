import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";

export const updateOverride = rateLimitedProcedure(ratelimit.update)
  .input(
    z.object({
      id: z.string(),
      limit: z.number(),
      duration: z.number(),
      async: z.boolean().nullable(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const override = await db.query.ratelimitOverrides
      .findFirst({
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
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update this override for this namespae. Please contact support using support@unkey.dev",
        });
      });

    if (!override || override.namespace.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({
        message:
          "We are unable to find the correct override. Please contact support using support@unkey.dev.",
        code: "NOT_FOUND",
      });
    }

    await db.transaction(async (tx) => {
      await tx
        .update(schema.ratelimitOverrides)
        .set({
          limit: input.limit,
          duration: input.duration,
          updatedAt: new Date(),
          async: input.async,
        })
        .where(eq(schema.ratelimitOverrides.id, override.id))
        .catch((_err) => {
          throw new TRPCError({
            message:
              "We are unable to update the override. Please contact support using support@unkey.dev.",
            code: "INTERNAL_SERVER_ERROR",
          });
        });
      await insertAuditLogs(tx, {
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
      }).catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update the override. Please contact support using support@unkey.dev",
        });
      });
    });
  });
