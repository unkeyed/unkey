import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { auth, t } from "../../trpc";
import { createBudgetInputSchema } from "./create";

export const updateBudget = t.procedure
  .use(auth)
  .input(
    createBudgetInputSchema.extend({
      budgetId: z.string(),
      enabled: z.boolean(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const budget = await db.query.budgets.findFirst({
      where: (table, { eq }) => eq(table.id, input.budgetId),
      with: {
        workspace: true,
      },
    });
    if (!budget) {
      throw new TRPCError({ message: "budget not found", code: "NOT_FOUND" });
    }
    if (budget.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({ message: "budget not found", code: "NOT_FOUND" });
    }

    await db
      .update(schema.budgets)
      .set({
        enabled: input.enabled,
        name: input.name,
        type: input.type,
        fixedAmount: input.fixedAmount,
        data: {
          additionalEmails: input.additionalEmails,
        },
      })
      .where(eq(schema.budgets.id, input.budgetId));
  });
