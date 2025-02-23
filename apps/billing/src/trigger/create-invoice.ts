import { clickhouse } from "@/lib/clickhouse";
import {
  type FixedSubscription,
  type TieredSubscription,
  calculateTieredPrices,
} from "@unkey/billing";

import { env } from "@/lib/env";
import { logger, task } from "@trigger.dev/sdk/v3";

import { connectDatabase } from "@/lib/db";

import Stripe from "stripe";

type Payload = {
  workspaceId: string;
  year: number;
  month: number;
};

export const createInvoiceTask = task({
  id: "billing_invoicing_createInvoice",
  run: async (payload: Payload) => {
    const { workspaceId, year, month } = payload;

    const db = connectDatabase();
    const stripe = new Stripe(env().STRIPE_SECRET_KEY, {
      apiVersion: "2023-10-16",
      typescript: true,
    });

    const workspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.id, workspaceId), isNull(table.deletedAtM)),
    });

    if (!workspace) {
      throw new Error(`workspace ${workspaceId} not found`);
    }

    if (!workspace.stripeCustomerId) {
      throw new Error(`workspace ${workspaceId} has no stripe customer id`);
    }

    const paymentMethods = await stripe.paymentMethods.list({
      customer: workspace.stripeCustomerId!,
      limit: 1,
    });

    const paymentMethodId = paymentMethods.data.at(0)?.id;

    const invoice = await stripe.invoices.create({
      customer: workspace.stripeCustomerId!,
      default_payment_method: paymentMethodId,
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

    let prorate: number | undefined = undefined;
    if (
      workspace.planChanged &&
      new Date(workspace.planChanged).getUTCFullYear() === year &&
      new Date(workspace.planChanged).getUTCMonth() + 1 === month
    ) {
      const start = new Date(year, month - 1, 1);
      const end = new Date(year, month, 1);
      prorate =
        (end.getTime() - new Date(workspace.planChanged).getTime()) /
        (end.getTime() - start.getTime());
      logger.info("prorating", { start, end, prorate });
    } else if (
      workspace.createdAt &&
      new Date(workspace.createdAt).getUTCFullYear() === year &&
      new Date(workspace.createdAt).getUTCMonth() + 1 === month
    ) {
      const start = new Date(year, month - 1, 1);
      const end = new Date(year, month, 1);
      prorate =
        (end.getTime() - new Date(workspace.createdAt).getTime()) /
        (end.getTime() - start.getTime());
      logger.info("prorating", { start, end, prorate });
    }

    if (workspace.subscriptions?.plan) {
      await createFixedCostInvoiceItem({
        stripe,
        invoiceId: invoice.id,
        stripeCustomerId: workspace.stripeCustomerId!,
        name: "Pro plan",
        sub: workspace.subscriptions.plan,
        prorate,
      });
    }

    /**
     * Verifications
     */
    if (workspace.subscriptions?.verifications) {
      const verifications = await clickhouse.billing.billableVerifications({
        workspaceId: workspace.id,
        year,
        month,
      });

      await createTieredInvoiceItem({
        stripe,
        invoiceId: invoice.id,
        stripeCustomerId: workspace.stripeCustomerId!,
        name: "Verifications",
        sub: workspace.subscriptions.verifications,
        usage: verifications,
      });
    }

    /**
     * Ratelimits
     */
    if (workspace.subscriptions?.ratelimits) {
      const ratelimits = await clickhouse.billing.billableRatelimits({
        workspaceId: workspace.id,
        year,
        month,
      });

      await createTieredInvoiceItem({
        stripe,
        invoiceId: invoice.id,
        stripeCustomerId: workspace.stripeCustomerId!,
        name: "Ratelimits",
        sub: workspace.subscriptions.ratelimits,
        usage: ratelimits,
      });
    }

    /**
     * Support
     */
    if (workspace.subscriptions?.support) {
      await createFixedCostInvoiceItem({
        stripe,
        invoiceId: invoice.id,
        stripeCustomerId: workspace.stripeCustomerId!,
        name: "Professional Support",
        sub: workspace.subscriptions.support,
        prorate,
      });
    }

    return {
      invoiceId: invoice.id,
    };
  },
});

async function createFixedCostInvoiceItem({
  stripe,
  invoiceId,
  stripeCustomerId,
  name,
  sub,
  prorate,
}: {
  stripe: Stripe;
  invoiceId: string;
  stripeCustomerId: string;
  name: string;
  sub: FixedSubscription;
  /**
   * number between 0 and 1 to indicate how much to charge
   * if they have had a fixed cost item for 15/30 days, this should be 0.5
   */
  prorate?: number;
}): Promise<void> {
  await stripe.invoiceItems.create({
    customer: stripeCustomerId,
    invoice: invoiceId,
    quantity: 1,
    price_data: {
      currency: "usd",
      product: sub.productId,
      unit_amount_decimal:
        typeof prorate === "number" ? (Number.parseInt(sub.cents) * prorate).toFixed(2) : sub.cents,
    },
    currency: "usd",
    description: typeof prorate === "number" ? `${name} (Prorated)` : name,
  });
}

async function createTieredInvoiceItem({
  stripe,
  invoiceId,

  stripeCustomerId,
  name,
  sub,
  usage,
}: {
  stripe: Stripe;
  invoiceId: string;

  stripeCustomerId: string;
  name: string;
  sub: TieredSubscription;
  usage: number;
}): Promise<void> {
  const cost = calculateTieredPrices(sub.tiers, usage);
  if (cost.err) {
    throw new Error(cost.err.message);
  }

  for (const tier of cost.val.tiers) {
    if (tier.quantity > 0 && tier.centsPerUnit) {
      const description = `${name} ${tier.firstUnit}${tier.lastUnit ? `-${tier.lastUnit}` : "+"}`;
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
    }
  }
}
