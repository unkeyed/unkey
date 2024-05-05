import { db, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const createUsageExport = t.procedure
  .use(auth)
  .input(
    z.object({
      interval: z
        .number()
        .min(60 * 1000)
        .max(30 * 24 * 60 * 60 * 1000),
      webhookId: z.string(),
      keySpaceId: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const ws = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      with: {
        keySpaces: { where: (table, { eq }) => eq(table.id, input.keySpaceId) },
        webhooks: { where: (table, { eq }) => eq(table.id, input.webhookId) },
      },
    });
    if (!ws) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "workspace not found",
      });
    }
    if (ws.keySpaces.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "keyspace not found",
      });
    }
    if (ws.webhooks.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "webhook not found",
      });
    }

    const reporterId = newId("reporter");

    await db.insert(schema.usageReporters).values({
      id: reporterId,
      createdAt: new Date(),
      interval: input.interval,
      keySpaceId: input.keySpaceId,
      nextExecution: 0,
      webhookId: input.webhookId,
      workspaceId: ws.id,
    });

    await ingestAuditLogs({
      workspaceId: ws.id,
      actor: { type: "user", id: ctx.user.id },
      event: "reporter.create",
      description: `Created ${reporterId}`,
      resources: [
        {
          type: "reporter",
          id: reporterId,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });

    return {
      id: reporterId,
    };
  });
