import { and, db, eq, gte, lte, schema } from "@/lib/db";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

// Get revenue analytics
export const getRevenueAnalytics = workspaceProcedure
  .input(
    z.object({
      startDate: z.number(),
      endDate: z.number(),
      granularity: z.enum(["day", "week", "month"]).default("day"),
    }),
  )
  .query(async ({ ctx, input }) => {
    const invoices = await db.query.billingInvoices.findMany({
      where: and(
        eq(schema.billingInvoices.workspaceId, ctx.workspace.id),
        eq(schema.billingInvoices.status, "paid"),
        gte(schema.billingInvoices.billingPeriodStart, input.startDate),
        lte(schema.billingInvoices.billingPeriodEnd, input.endDate),
      ),
      columns: {
        billingPeriodEnd: true,
        totalAmount: true,
      },
    });

    // Group by date based on granularity
    const dataPoints: Record<string, { revenue: number; invoiceCount: number }> = {};

    for (const invoice of invoices) {
      const date = new Date(invoice.billingPeriodEnd);
      let key: string;

      switch (input.granularity) {
        case "day":
          key = date.toISOString().split("T")[0];
          break;
        case "week": {
          const weekStart = new Date(date);
          weekStart.setDate(date.getDate() - date.getDay());
          key = weekStart.toISOString().split("T")[0];
          break;
        }
        case "month":
          key = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
          break;
      }

      if (!dataPoints[key]) {
        dataPoints[key] = { revenue: 0, invoiceCount: 0 };
      }
      dataPoints[key].revenue += invoice.totalAmount;
      dataPoints[key].invoiceCount += 1;
    }

    return Object.entries(dataPoints)
      .map(([date, data]) => ({
        date,
        revenue: data.revenue,
        invoiceCount: data.invoiceCount,
      }))
      .sort((a, b) => a.date.localeCompare(b.date));
  });

// Get usage analytics
export const getUsageAnalytics = workspaceProcedure
  .input(
    z.object({
      startDate: z.number(),
      endDate: z.number(),
    }),
  )
  .query(async ({ ctx, input }) => {
    const invoices = await db.query.billingInvoices.findMany({
      where: and(
        eq(schema.billingInvoices.workspaceId, ctx.workspace.id),
        gte(schema.billingInvoices.billingPeriodStart, input.startDate),
        lte(schema.billingInvoices.billingPeriodEnd, input.endDate),
      ),
      columns: {
        verificationCount: true,
        ratelimitCount: true,
      },
    });

    const totalVerifications = invoices.reduce((sum, i) => sum + i.verificationCount, 0);
    const totalRatelimits = invoices.reduce((sum, i) => sum + i.ratelimitCount, 0);

    return {
      totalVerifications,
      totalRatelimits,
      invoiceCount: invoices.length,
    };
  });

// Export billing data as CSV
export const exportBillingData = workspaceProcedure
  .input(
    z.object({
      startDate: z.number().optional(),
      endDate: z.number().optional(),
    }),
  )
  .query(async ({ ctx, input }) => {
    const conditions = [eq(schema.billingInvoices.workspaceId, ctx.workspace.id)];

    if (input.startDate) {
      conditions.push(gte(schema.billingInvoices.billingPeriodStart, input.startDate));
    }

    if (input.endDate) {
      conditions.push(lte(schema.billingInvoices.billingPeriodEnd, input.endDate));
    }

    const invoices = await db.query.billingInvoices.findMany({
      where: and(...conditions),
      with: {
        endUser: true,
      },
      orderBy: (invoices, { desc }) => [desc(invoices.createdAtM)],
    });

    // Generate CSV content
    const headers = [
      "Invoice ID",
      "End User ID",
      "External ID",
      "Email",
      "Billing Period Start",
      "Billing Period End",
      "Verifications",
      "Rate Limits",
      "Total Amount",
      "Currency",
      "Status",
      "Created At",
    ];

    const rows = invoices.map((invoice) => [
      invoice.id,
      invoice.endUserId,
      invoice.endUser?.externalId ?? "",
      invoice.endUser?.email ?? "",
      new Date(invoice.billingPeriodStart).toISOString(),
      new Date(invoice.billingPeriodEnd).toISOString(),
      invoice.verificationCount.toString(),
      invoice.ratelimitCount.toString(),
      (invoice.totalAmount / 100).toFixed(2),
      invoice.currency,
      invoice.status,
      new Date(invoice.createdAtM).toISOString(),
    ]);

    const csv = [headers.join(","), ...rows.map((row) => row.join(","))].join("\n");

    return { csv, filename: `billing-export-${Date.now()}.csv` };
  });
