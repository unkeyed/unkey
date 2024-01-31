import { Tinybird } from "@/lib/tinybird";
import {
  type FixedSubscription,
  type TieredSubscription,
  calculateTieredPrices,
} from "@unkey/billing";
import { z } from "zod";

import { env } from "@/lib/env";
import { client } from "@/trigger";

import { connectDatabase } from "@/lib/db";
import { type IO, eventTrigger } from "@trigger.dev/sdk";

import Stripe from "stripe";

export const createInvoiceJob = client.defineJob({
  id: "billing.invoicing.createInvoice",
  name: "Collect usage and create invoice",
  version: "0.0.2",
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
        where: (table, { and, eq, isNull }) =>
          and(eq(table.id, workspaceId), isNull(table.deletedAt)),
      }),
    );

    if (!workspace) {
      throw new Error(`workspace ${workspaceId} not found`);
    }

    if (!workspace.stripeCustomerId) {
      throw new Error(`workspace ${workspaceId} has no stripe customer id`);
    }

    const invoiceId = await io.runTask(`create invoice for ${workspace.id}`, async () =>
      stripe.invoices
        .create({
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
        })
        .then((invoice) => invoice.id),
    );

    let prorate: number | undefined = undefined;
    if (
      workspace.planChanged &&
      workspace.planChanged.getUTCFullYear() === year &&
      workspace.planChanged.getUTCMonth() === month
    ) {
      const start = new Date(year, month, 1);
      const end = new Date(year, month + 1, 1);
      prorate =
        (workspace.planChanged.getTime() - start.getTime()) / (end.getTime() - start.getTime());
    }

    if (workspace.subscriptions?.plan) {
      await createFixedCostInvoiceItem({
        stripe,
        invoiceId,
        stripeCustomerId: workspace.stripeCustomerId!,
        io,
        name: "Pro plan",
        sub: workspace.subscriptions.plan,
        prorate,
      });
    }

    /**
     * Active keys
     */
    if (workspace.subscriptions?.activeKeys) {
      const activeKeys = await io.runTask(`get active keys for ${workspace.id}`, async () =>
        tinybird
          .activeKeys({
            workspaceId: workspace.id,
            year,
            month,
          })
          .then((res) => res.data.at(0)?.keys ?? 0),
      );

      await createTieredInvoiceItem({
        stripe,
        invoiceId,
        stripeCustomerId: workspace.stripeCustomerId!,
        io,
        name: "Active Keys",
        sub: workspace.subscriptions.activeKeys,
        usage: activeKeys,
      });
    }

    /**
     * Verifications
     */
    if (workspace.subscriptions?.verifications) {
      const verifications = await io.runTask(`get verifications for ${workspace.id}`, async () =>
        tinybird
          .verifications({
            workspaceId: workspace.id,
            year,
            month,
          })
          .then((res) => res.data.at(0)?.success ?? 0),
      );

      await createTieredInvoiceItem({
        stripe,
        invoiceId,
        stripeCustomerId: workspace.stripeCustomerId!,
        io,
        name: "Verifications",
        sub: workspace.subscriptions.verifications,
        usage: verifications,
      });
    }

    /**
     * Support
     */
    if (workspace.subscriptions?.support) {
      await createFixedCostInvoiceItem({
        stripe,
        invoiceId,
        stripeCustomerId: workspace.stripeCustomerId!,
        io,
        name: "Professional Support",
        sub: workspace.subscriptions.support,
        prorate,
      });
    }

    return {
      invoiceId,
    };
  },
});

async function createFixedCostInvoiceItem({
  stripe,
  invoiceId,
  io,
  stripeCustomerId,
  name,
  sub,
  prorate,
}: {
  stripe: Stripe;
  invoiceId: string;
  io: IO;
  stripeCustomerId: string;
  name: string;
  sub: FixedSubscription;
  /**
   * number between 0 and 1 to indicate how much to charge
   * if they have had a fixed cost item for 15/30 days, this should be 0.5
   */
  prorate?: number;
}): Promise<void> {
  await io.runTask(name, async () => {
    await stripe.invoiceItems.create({
      customer: stripeCustomerId,
      invoice: invoiceId,
      quantity: 1,
      price_data: {
        currency: "usd",
        product: sub.productId,
        unit_amount_decimal:
          typeof prorate === "number" ? (parseInt(sub.cents) * prorate).toFixed(2) : sub.cents,
      },
      currency: "usd",
      description: typeof prorate === "number" ? `${name} (Prorated)` : name,
    });
  });
}

async function createTieredInvoiceItem({
  stripe,
  invoiceId,
  io,
  stripeCustomerId,
  name,
  sub,
  usage,
}: {
  stripe: Stripe;
  invoiceId: string;
  io: IO;
  stripeCustomerId: string;
  name: string;
  sub: TieredSubscription;
  usage: number;
}): Promise<void> {
  const cost = calculateTieredPrices(sub.tiers, usage);
  if (cost.error) {
    throw new Error(cost.error.message);
  }

  for (const tier of cost.value.tiers) {
    if (tier.quantity > 0 && tier.centsPerUnit) {
      const description = `${name} ${tier.firstUnit}${tier.lastUnit ? `-${tier.lastUnit}` : "+"}`;
      await io.runTask(description, async () => {
        await stripe.invoiceItems.create({
          customer: stripeCustomerId,
          invoice: invoiceId,
          quantity: tier.quantity,
          price_data: {
            currency: "usd",
            product: sub.productId,
            unit_amount_decimal: tier.centsPerUnit!,
          },
          currency: "usd",
          description,
        });
      });
    }
  }
}
