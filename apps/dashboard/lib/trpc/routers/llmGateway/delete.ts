import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { auth, t } from "../../trpc";

export const deleteLlmGateway = t.procedure
  .use(auth)
  .input(z.object({ gatewayId: z.string() }))
  .mutation(async ({ ctx, input }) => {
    const llmGateway = await db.query.llmGateways.findFirst({
      where: (table, { eq, and }) => and(eq(table.id, input.gatewayId)),
      with: {
        workspace: {
          columns: {
            id: true,
            tenantId: true,
          },
        },
      },
    });

    if (!llmGateway || llmGateway.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({ code: "NOT_FOUND", message: "LLM gateway not found" });
    }

    await db.transaction(async (tx) => {
      await tx.delete(schema.llmGateways).where(eq(schema.llmGateways.id, input.gatewayId));
    });

    await ingestAuditLogs({
      workspaceId: llmGateway.workspace.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "llmGateway.delete",
      description: `Deleted ${llmGateway.id}`,
      resources: [
        {
          type: "gateway",
          id: llmGateway.id,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });

    return {
      id: llmGateway.id,
    };
  });
