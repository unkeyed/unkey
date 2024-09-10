import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { rateLimitedProcedure } from "../../trpc";
import { UPDATE_LIMIT_DURATION, UPDATE_LIMIT } from "@/lib/ratelimitValues";

export const toggleWebhook = rateLimitedProcedure({limit: UPDATE_LIMIT, duration: UPDATE_LIMIT_DURATION })
  .input(
    z.object({
      webhookId: z.string(),
      enabled: z.boolean(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const ws = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      with: {
        webhooks: {
          where: (table, { eq }) => eq(table.id, input.webhookId),
        },
      },
    });
    if (!ws) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct workspace. Please contact support using support@unkey.dev.",
      });
    }

    if (ws.webhooks.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct webhook. Please contact support using support@unkey.dev.",
      });
    }

    await db
      .update(schema.webhooks)
      .set({
        enabled: input.enabled,
      })
      .where(eq(schema.webhooks.id, input.webhookId))
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update the webhook. Please contact support using support@unkey.dev",
        });
      });

    await ingestAuditLogs({
      workspaceId: ws.id,
      actor: { type: "user", id: ctx.user.id },
      event: "webhook.update",
      description: `${input.enabled ? "Enabled" : "Disabled"} ${input.webhookId}`,
      resources: [{ type: "webhook", id: input.webhookId }],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });

    return {
      enabled: input.enabled,
    };
  });
