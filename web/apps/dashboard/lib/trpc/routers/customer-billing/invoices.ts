import { and, db, eq, gte, lte, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import { invoiceStatusSchema } from "./types";

// List invoices
export const listInvoices = workspaceProcedure
  .input(
    z
      .object({
        endUserId: z.string().optional(),
        status: invoiceStatusSchema.optional(),
        startDate: z.number().optional(),
        endDate: z.number().optional(),
        limit: z.number().min(1).max(100).default(50),
        offset: z.number().min(0).default(0),
      })
      .optional(),
  )
  .query(async ({ ctx, input }) => {
    const limit = input?.limit ?? 50;
    const offset = input?.offset ?? 0;

    // Build where conditions
    const conditions = [eq(schema.billingInvoices.workspaceId, ctx.workspace.id)];

    if (input?.endUserId) {
      conditions.push(eq(schema.billingInvoices.endUserId, input.endUserId));
    }

    if (input?.status) {
      conditions.push(eq(schema.billingInvoices.status, input.status));
    }

    if (input?.startDate) {
      conditions.push(gte(schema.billingInvoices.billingPeriodStart, input.startDate));
    }

    if (input?.endDate) {
      conditions.push(lte(schema.billingInvoices.billingPeriodEnd, input.endDate));
    }

    const invoices = await db.query.billingInvoices.findMany({
      where: and(...conditions),
      with: {
        endUser: true,
      },
      orderBy: (invoices, { desc }) => [desc(invoices.createdAtM)],
      limit,
      offset,
    });

    return invoices;
  });

// Get invoice by ID
export const getInvoice = workspaceProcedure
  .input(z.object({ id: z.string() }))
  .query(async ({ ctx, input }) => {
    const invoice = await db.query.billingInvoices.findFirst({
      where: eq(schema.billingInvoices.id, input.id),
      with: {
        endUser: {
          with: {
            pricingModel: true,
          },
        },
        transactions: {
          orderBy: (transactions, { desc }) => [desc(transactions.createdAtM)],
        },
      },
    });

    if (!invoice || invoice.workspaceId !== ctx.workspace.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Invoice not found",
      });
    }

    return invoice;
  });

// Get invoice summary stats
export const getInvoiceSummary = workspaceProcedure.query(async ({ ctx }) => {
  const invoices = await db.query.billingInvoices.findMany({
    where: eq(schema.billingInvoices.workspaceId, ctx.workspace.id),
    columns: {
      status: true,
      totalAmount: true,
    },
  });

  if (invoices.length === 0) {
    return {
      totalRevenue: 0,
      pendingRevenue: 0,
      totalInvoices: 0,
      statusCounts: {
        draft: 0,
        open: 0,
        paid: 0,
        void: 0,
        uncollectible: 0,
      },
    };
  }

  const totalRevenue = invoices
    .filter((i) => i.status === "paid")
    .reduce((sum, i) => sum + i.totalAmount, 0);

  const pendingRevenue = invoices
    .filter((i) => i.status === "open" || i.status === "draft")
    .reduce((sum, i) => sum + i.totalAmount, 0);

  const statusCounts = {
    draft: invoices.filter((i) => i.status === "draft").length,
    open: invoices.filter((i) => i.status === "open").length,
    paid: invoices.filter((i) => i.status === "paid").length,
    void: invoices.filter((i) => i.status === "void").length,
    uncollectible: invoices.filter((i) => i.status === "uncollectible").length,
  };

  return {
    totalRevenue,
    pendingRevenue,
    totalInvoices: invoices.length,
    statusCounts,
  };
});
