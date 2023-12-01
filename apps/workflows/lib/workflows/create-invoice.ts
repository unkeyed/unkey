import { connectDatabase } from "@/lib/db";
import { env } from "@/lib/env";
import { inngest } from "@/lib/inngest";
import { Tinybird } from "@/lib/tinybird";
import Stripe from "stripe";
import { z } from "zod";
const _metaSchema = z.object({
  priceIdActiveKeys: z.string().optional(),
  priceIdVerifications: z.string().optional(),
  priceIdSupport: z.string().optional(),
});

export const createInvoice = inngest.createFunction(
  { id: "billing/create.invoice" },
  { event: "billing/create.invoice" },
  async ({ event, step }) => {
    const { workspaceId, year, month } = event.data;

    const db = connectDatabase();
    const stripe = new Stripe(env().STRIPE_SECRET_KEY, {
      apiVersion: "2022-11-15",
      typescript: true,
    });
    const tinybird = new Tinybird(env().TINYBIRD_TOKEN);

    const workspace = await step.run(`get workspace ${workspaceId}`, async () =>
      db.query.workspaces.findFirst({
        where: (table, { eq }) => eq(table.id, workspaceId),
      }),
    );

    if (!workspace) {
      throw new Error(`workspace ${workspaceId} not found`);
    }

    if (!workspace.stripeCustomerId) {
      throw new Error(`workspace ${workspaceId} has no stripe customer id`);
    }

    const invoice = await stripe.invoices.create({
      customer: workspace.stripeCustomerId,
      auto_advance: false,
      metadata: {
        workspaceId: workspace.id,
        plan: workspace.plan,
        period: new Date(year, month - 1, 1).toLocaleString("en-US", {
          month: "long",
          year: "numeric",
        }),
      },
    });

    if (workspace.subscriptions?.plan?.priceId) {
      await stripe.invoiceItems.create({
        customer: workspace.stripeCustomerId,
        invoice: invoice.id,
        price: workspace.subscriptions.plan.priceId,
        currency: "USD",
        description: "Pro plan",
      });
    }

    if (workspace.subscriptions?.activeKeys?.priceId) {
      const activeKeys = await step.run(`get active keys for ${workspace.id}`, async () =>
        tinybird
          .activeKeys({
            workspaceId: workspace.id,
            year: event.data.year,
            month: event.data.month,
          })
          .then((res) => res.data.at(0)?.keys ?? 0),
      );
      await stripe.invoiceItems.create({
        customer: workspace.stripeCustomerId,
        invoice: invoice.id,
        quantity: activeKeys,
        price: workspace.subscriptions.activeKeys.priceId,
        currency: "USD",
        description: "Active keys",
      });
    }

    if (workspace.subscriptions?.verifications?.priceId) {
      const verifications = await step.run(`get verifications for ${workspace.id}`, async () =>
        tinybird
          .verifications({
            workspaceId: workspace.id,
            year: event.data.year,
            month: event.data.month,
          })
          .then((res) => res.data.at(0)?.success ?? 0),
      );

      await stripe.invoiceItems.create({
        customer: workspace.stripeCustomerId,
        invoice: invoice.id,
        quantity: verifications,
        price: workspace.subscriptions.verifications.priceId,
        currency: "USD",
        description: "Verifications",
      });
    }

    if (workspace.subscriptions?.support?.priceId) {
      await stripe.invoiceItems.create({
        customer: workspace.stripeCustomerId,
        invoice: invoice.id,
        price: workspace.subscriptions.support.priceId,
        currency: "USD",
        description: "Dedicated support",
      });
    }

    return {
      event,
      body: {
        invoiceId: invoice.id,
      },
    };
  },
);
