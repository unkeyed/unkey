import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { auth, t } from "../../trpc";

export const deleteBudget = t.procedure
  .use(auth)
  .input(
    z.object({
      budgetId: z.string(),
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

    await db.delete(schema.budgets).where(eq(schema.budgets.id, input.budgetId));
  });
