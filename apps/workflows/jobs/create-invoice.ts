import { Tinybird } from "@/lib/tinybird";
import Stripe from "stripe";
import { z } from "zod";

import { env } from "@/lib/env";
import { client } from "@/trigger";

import { connectDatabase } from "@/lib/db";
import { eventTrigger } from "@trigger.dev/sdk";

client.defineJob({
  id: "billing.invoicing.createInvoice",
  name: "Collect usage and create invoice",
  version: "0.0.1",
  trigger: eventTrigger({
    name: "billing.invoicing.createInvoice",
    schema: z.object({
      workspaceId: z.string(),
      year: z.number(),
      month: z.number(),
    }),
  }),

  run: async (payload, io, _ctx) => {
    const { workspaceId, year, month } = payload;

    const db = connectDatabase();
    const stripe = new Stripe(env().STRIPE_SECRET_KEY, {
      apiVersion: "2022-11-15",
      typescript: true,
    });
    const tinybird = new Tinybird(env().TINYBIRD_TOKEN);

    const workspace = await io.runTask(`get workspace ${workspaceId}`, async () =>
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

    const invoice = await io.runTask(`create invoice for ${workspace.id}`, async () => {
      const inv = await stripe.invoices.create({
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
      });
      return {
        id: inv.id,
      };
    });

    /**
     * Plan
     */
    if (workspace.subscriptions?.plan) {
      await io.runTask("add plan", async () => {
        await stripe.invoiceItems.create({
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
        });
      });
    }

    /**
     * Active keys
     */
    if (workspace.subscriptions?.activeKeys) {
      let activeKeys = await io.runTask(`get active keys for ${workspace.id}`, async () =>
        tinybird
          .activeKeys({
            workspaceId: workspace.id,
            year,
            month,
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
          await io.runTask(`add active keys tier ${tier.firstUnit}-${tier.lastUnit}`, async () => {
            await stripe.invoiceItems.create({
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
            });
          });
        }
      }
    }
    /**
     * Verifications
     */
    if (workspace.subscriptions?.verifications) {
      let verifications = await io.runTask(`get verifications for ${workspace.id}`, async () =>
        tinybird
          .verifications({
            workspaceId: workspace.id,
            year,
            month,
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
          await io.runTask(`add verification tier ${tier.firstUnit}-${tier.lastUnit}`, async () => {
            await stripe.invoiceItems.create({
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
            });
          });
        }
      }
    }

    /**
     * Support
     */
    if (workspace.subscriptions?.support) {
      await io.runTask("add support", async () => {
        await stripe.invoiceItems.create({
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
        });
      });
    }

    return {
      invoiceId: invoice.id,
    };
  },
});
