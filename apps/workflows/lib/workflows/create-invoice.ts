import { connectDatabase, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { inngest } from "@/lib/inngest";
import { Tinybird } from "@/lib/tinybird";
import Stripe from "stripe";
import { z } from "zod";
const metaSchema = z.object({
  priceIdActiveKeys: z.string().optional(),
  priceIdVerifications: z.string().optional(),
  priceIdSupport: z.string().optional(),
})

export const createInvoice = inngest.createFunction(
  { id: "billing/create.invoice" },
  { event: "invoice.draft" },
  async ({ event, step, logger }) => {

    const { workspace } = event.data

    const db = connectDatabase();
    const stripe = new Stripe(env().STRIPE_SECRET_KEY, {
      apiVersion: "2022-11-15",
      typescript: true,
    });
    const tinybird = new Tinybird(env().TINYBIRD_TOKEN);




    const invoice = await stripe.invoices.create({
      customer: event.data.stripeCustomerId,
      auto_advance: false,
      metadata: {
        workspaceId: workspace.Id,
        plan: event.data.plan,
        period: new Date(event.data.year, event.data.month - 1, 1).toLocaleString("en-US", { month: "long", year: "numeric" })
      },
    });

    if (workspace.subscriptions?.priceIdPlan) {
      await stripe.invoiceItems.create({
        customer: event.data.stripeCustomerId,
        invoice: invoice.id,
        price: workspace.subscriptions.priceIdPlan,
        currency: "USD",
        description: `Dedicated support`,
      })
    }

    if (workspace.subscriptions?.priceIdActiveKeys) {
      const activeKeys = await step.run(`get active keys for ${workspace.id}`, async () =>
        tinybird.activeKeys({
          workspaceId: workspace.id,
          year: event.data.year,
          month: event.data.month,
        }).then((res) => res.data.at(0)?.keys ?? 0),
      );
      await stripe.invoiceItems.create({
        customer: event.data.stripeCustomerId,
        invoice: invoice.id,
        quantity: activeKeys,
        price: workspace.subscriptions.priceIdActiveKeys,
        currency: "USD",
        description: `Active keys`,
      })
    }


    if (workspace.subscriptions?.priceIdVerifications) {
      const verifications = await step.run(`get verifications for ${workspace.id}`, async () =>
        tinybird.verifications({
          workspaceId: workspace.id,
          year: event.data.year,
          month: event.data.month,
        }).then((res) => res.data.at(0)?.success ?? 0),
      );


      await stripe.invoiceItems.create({
        customer: event.data.stripeCustomerId,
        invoice: invoice.id,
        quantity: verifications,
        price: workspace.subscriptions.priceIdVerifications,
        currency: "USD",
        description: `Verifications`,
      })
    }

    if (workspace.subscriptions?.priceIdSupport) {
      await stripe.invoiceItems.create({
        customer: event.data.stripeCustomerId,
        invoice: invoice.id,
        price: workspace.subscriptions.priceIdSupport,
        currency: "USD",
        description: `Dedicated support`,
      })
    }


    return {
      event,
      body: {
        invoiceId: invoice.id,
      },
    };
  },
);
