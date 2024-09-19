import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const deleteWebhook = rateLimitedProcedure(ratelimit.delete)
  .input(
    z.object({
      webhookId: z.string(),
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
      .delete(schema.webhooks)
      .where(eq(schema.webhooks.id, input.webhookId))
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to delete the webhook. Please contact support using support@unkey.dev",
        });
      });

    await ingestAuditLogs({
      workspaceId: ws.id,
      actor: { type: "user", id: ctx.user.id },
      event: "webhook.delete",
      description: `Deleted ${input.webhookId}`,
      resources: [{ type: "webhook", id: input.webhookId }],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });

    return;
  });
