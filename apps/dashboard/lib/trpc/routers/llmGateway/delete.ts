import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { db, eq, schema } from "@/lib/db";
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

    return {
      id: llmGateway.id,
    };
  });
