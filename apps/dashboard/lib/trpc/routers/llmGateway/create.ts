import { db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const createLlmGateway = t.procedure
  .use(auth)
  .input(
    z.object({
      subdomain: z.string().min(1).max(50),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const ws = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
    });
    if (!ws) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "workspace not found",
      });
    }

    const llmGatewayId = newId("llmGateway");

    await db.insert(schema.llmGateways).values({
      id: llmGatewayId,
      subdomain: input.subdomain,
      name: input.subdomain,
      workspaceId: ws.id,
    });

    return {
      id: llmGatewayId,
    };
  });
