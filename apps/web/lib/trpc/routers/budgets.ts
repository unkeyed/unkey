import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { auth, t } from "../trpc";

const createInputSchema = z.object({
  name: z.string().optional(),
  type: z.enum(["soft", "hard"]).default("soft"),
  fixedAmount: z.number().positive(),
  additionalEmails: z.array(z.string().email()).max(10).optional(),
});

export const budgetRouter = t.router({
  create: t.procedure
    .use(auth)
    .input(createInputSchema)
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
    }),
  edit: t.procedure
    .use(auth)
    .input(
      createInputSchema.extend({
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
    }),
  delete: t.procedure
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
    }),
});
