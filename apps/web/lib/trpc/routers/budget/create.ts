import { db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";

import { auth, t } from "../../trpc";

export const createBudgetInputSchema = z.object({
  name: z.string().optional(),
  type: z.enum(["soft", "hard"]).default("soft"),
  fixedAmount: z.number().positive(),
  additionalEmails: z.array(z.string().email()).max(10).optional(),
});

export const createBudget = t.procedure
  .use(auth)
  .input(createBudgetInputSchema)
  .mutation(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
    });
    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "workspace not found",
      });
    }

    const budgetId = newId("budget");

    await db.insert(schema.budgets).values({
      id: budgetId,
      name: input.name,
      type: input.type,
      fixedAmount: input.fixedAmount,
      data: {
        additionalEmails: input.additionalEmails,
      },
      workspaceId: workspace.id,
      createdAt: new Date(),
    });
  });
