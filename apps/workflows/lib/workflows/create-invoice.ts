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

    const invoice = await step.run("create invoice", () =>
      stripe.invoices.create({
        customer: workspace.stripeCustomerId!,
        auto_advance: false,
        custom_fields: [
          {
            name: "Workspace",
            value: workspace.name,
          },
          {
            name: "Billing Period",
            value: new Date(year, month - 1, 1).toLocaleString("en-US", {
              month: "long",
              year: "numeric",
            }),
          },
        ],
      }),
    );

    /**
     * Plan
     */
    if (workspace.subscriptions?.plan) {
      await step.run("add plan", () =>
        stripe.invoiceItems.create({
          customer: workspace.stripeCustomerId!,
          invoice: invoice.id,
          quantity: 1,
          price_data: {
            currency: "usd",
            product: workspace.subscriptions!.plan!.productId,
            unit_amount: workspace.subscriptions!.plan!.price * 100,
          },
          currency: "usd",
          description: "Pro Plan",
        }),
      );
    }

    /**
     * Active keys
     */
    if (workspace.subscriptions?.activeKeys) {
      let activeKeys = await step.run(`get active keys for ${workspace.id}`, async () =>
        tinybird
          .activeKeys({
            workspaceId: workspace.id,
            year: event.data.year,
            month: event.data.month,
          })
          .then((res) => res.data.at(0)?.keys ?? 0),
      );

      for (const tier of workspace.subscriptions!.activeKeys!.tiers) {
        if (activeKeys <= 0) {
          break;
        }

        const quantity =
          tier.lastUnit === null
            ? activeKeys
            : Math.min(tier.lastUnit - tier.firstUnit + 1, activeKeys);
        activeKeys -= quantity;
        if (quantity > 0) {
          await step.run(`add active keys tier ${tier.firstUnit}-${tier.lastUnit}`, () =>
            stripe.invoiceItems.create({
              customer: workspace.stripeCustomerId!,
              invoice: invoice.id,
              quantity,
              price_data: {
                currency: "usd",
                product: workspace.subscriptions!.activeKeys!.productId,
                unit_amount_decimal: (tier.perUnit * 100).toString(),
              },
              currency: "usd",
              description: `Active Keys ${tier.firstUnit}${
                tier.lastUnit ? `-${tier.lastUnit}` : "+"
              }`,
            }),
          );
        }
      }
    }
    /**
     * Verifications
     */
    if (workspace.subscriptions?.verifications) {
      let verifications = await step.run(`get verifications for ${workspace.id}`, async () =>
        tinybird
          .verifications({
            workspaceId: workspace.id,
            year: event.data.year,
            month: event.data.month,
          })
          .then((res) => res.data.at(0)?.success ?? 0),
      );

      for (const tier of workspace.subscriptions!.verifications!.tiers) {
        if (verifications <= 0) {
          break;
        }

        const quantity =
          tier.lastUnit === null
            ? verifications
            : Math.min(tier.lastUnit - tier.firstUnit + 1, verifications);
        verifications -= quantity;
        if (quantity > 0) {
          await step.run(`add verification tier ${tier.firstUnit}-${tier.lastUnit}`, () =>
            stripe.invoiceItems.create({
              customer: workspace.stripeCustomerId!,
              invoice: invoice.id,
              quantity,
              price_data: {
                currency: "usd",
                product: workspace.subscriptions!.verifications!.productId,

                unit_amount_decimal: (tier.perUnit * 100).toString(),
              },

              currency: "usd",
              description: `Verifications ${tier.firstUnit}${
                tier.lastUnit ? `-${tier.lastUnit}` : "+"
              }`,
            }),
          );
        }
      }
    }

    /**
     * Support
     */
    if (workspace.subscriptions?.support) {
      await step.run("add support", () =>
        stripe.invoiceItems.create({
          customer: workspace.stripeCustomerId!,
          invoice: invoice.id,
          quantity: 1,
          price_data: {
            currency: "usd",
            product: workspace.subscriptions!.support!.productId,
            unit_amount: workspace.subscriptions!.support!.price * 100,
          },
          currency: "usd",
          description: "Professional support",
        }),
      );
    }

    return {
      event,
      body: {
        invoiceId: invoice.id,
      },
    };
  },
);
