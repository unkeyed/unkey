import { WorkflowEntrypoint, type WorkflowEvent, type WorkflowStep } from "cloudflare:workers";
import {
  type FixedSubscription,
  type TieredSubscription,
  calculateTieredPrices,
} from "@unkey/billing";

import { ClickHouse } from "@unkey/clickhouse";
import Stripe from "stripe";
import { createConnection, eq, schema } from "../lib/db";
import type { Env } from "../lib/env";

// User-defined params passed to your workflow
// biome-ignore lint/complexity/noBannedTypes: we just don't have any params here
type Params = {};

// <docs-tag name="workflow-entrypoint">
export class Invoicing extends WorkflowEntrypoint<Env, Params> {
  async run(event: WorkflowEvent<Params>, step: WorkflowStep) {
    const db = createConnection({
      host: this.env.DATABASE_HOST,
      username: this.env.DATABASE_USERNAME,
      password: this.env.DATABASE_PASSWORD,
    });

    const clickhouse = new ClickHouse({ url: this.env.CLICKHOUSE_URL });

    const stripe = new Stripe(this.env.STRIPE_SECRET_KEY, {
      apiVersion: "2023-10-16",
      typescript: true,
    });
    let workspaces = await step.do("fetch workspaces", () =>
      db.query.workspaces.findMany({
        where: (table, { isNotNull, isNull, not, eq, and }) =>
          and(
            isNotNull(table.stripeCustomerId),
            isNotNull(table.subscriptions),
            not(eq(table.plan, "free")),
            isNull(table.deletedAtM),
          ),
      }),
    );

    // hack to filter out workspaces with `{}` as subscriptions
    workspaces = workspaces.filter(
      (ws) => ws.subscriptions && Object.keys(ws.subscriptions).length > 0,
    );

    console.info(`found ${workspaces.length} workspaces`);

    /**
     * Dates gymnastics to get the previous month's number, ie: if it's December now, it returns -> 11
     */
    let t = new Date();
    try {
      // cf stopped sending valid `Date` objects for some reason, so we fall back to Date.now()
      t = event.timestamp;
    } catch {}
    t.setUTCMonth(t.getUTCMonth() - 1);
    const year = t.getUTCFullYear();
    const month = t.getUTCMonth() + 1; // months are 0 indexed

    for (const workspace of workspaces) {
      if (!workspace.stripeCustomerId) {
        throw new Error(`workspace ${workspace.id} has no stripe customer id`);
      }

      const paymentMethodId = await step.do(`list payment method for ${workspace.id}`, async () => {
        const methods = await stripe.paymentMethods.list({
          customer: workspace.stripeCustomerId!,
          limit: 1,
        });
        const primaryMethod = methods.data.at(0);
        if (!primaryMethod) {
          throw new Error(`Workspace ${workspace.id} doesn't have a payment method`);
        }
        return primaryMethod.id;
      });

      const invoiceId = await step.do(`create invoice for ${workspace.id}`, async () => {
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

        return invoice.id;
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
        console.info("prorating", { start, end, prorate });
      } else if (
        workspace.createdAtM &&
        new Date(workspace.createdAtM).getUTCFullYear() === year &&
        new Date(workspace.createdAtM).getUTCMonth() + 1 === month
      ) {
        const start = new Date(year, month - 1, 1);
        const end = new Date(year, month, 1);
        prorate =
          (end.getTime() - new Date(workspace.createdAtM).getTime()) /
          (end.getTime() - start.getTime());
        console.info("prorating", { start, end, prorate });
      }

      if (workspace.subscriptions?.plan) {
        await step.do(`add pro plan to invoice for ${workspace.id}`, () =>
          createFixedCostInvoiceItem({
            stripe,
            invoiceId: invoiceId,
            stripeCustomerId: workspace.stripeCustomerId!,
            name: "Pro plan",
            sub: workspace.subscriptions!.plan!,
            prorate,
          }),
        );
      }

      /**
       * Verifications
       */
      if (workspace.subscriptions?.verifications) {
        const verifications = await step.do(`query verifications for ${workspace.id}`, () =>
          clickhouse.billing.billableVerifications({
            workspaceId: workspace.id,
            year,
            month,
          }),
        );

        await step.do(`add verifications to invoice ${invoiceId} for ${workspace.id}`, () =>
          createTieredInvoiceItem({
            stripe,
            invoiceId: invoiceId,
            stripeCustomerId: workspace.stripeCustomerId!,
            name: "Verifications",
            sub: workspace.subscriptions!.verifications!,
            usage: verifications,
          }),
        );
      }

      /**
       * Ratelimits
       */
      if (workspace.subscriptions?.ratelimits) {
        const ratelimits = await step.do(`query ratelimits for ${workspace.id}`, () =>
          clickhouse.billing.billableRatelimits({
            workspaceId: workspace.id,
            year,
            month,
          }),
        );

        await step.do(`add ratelimits to invoice ${invoiceId} for ${workspace.id}`, () =>
          createTieredInvoiceItem({
            stripe,
            invoiceId: invoiceId,
            stripeCustomerId: workspace.stripeCustomerId!,
            name: "Ratelimits",
            sub: workspace.subscriptions!.ratelimits!,
            usage: ratelimits,
          }),
        );
      }

      /**
       * Support
       */
      if (workspace.subscriptions?.support) {
        await step.do(`add support to invoice ${invoiceId} for ${workspace.id}`, () =>
          createFixedCostInvoiceItem({
            stripe,
            invoiceId: invoiceId,
            stripeCustomerId: workspace.stripeCustomerId!,
            name: "Professional Support",
            sub: workspace.subscriptions!.support!,
            prorate,
          }),
        );
      }
    }

    const workspacesWithDowngradeRequest = await step.do("fetch downgrade requests", () =>
      db.query.workspaces.findMany({
        where: (table, { and, isNotNull }) => and(isNotNull(table.planDowngradeRequest)),
      }),
    );

    for (const ws of workspacesWithDowngradeRequest) {
      await step.do(`downgrade ${ws.id}`, () =>
        db
          .update(schema.workspaces)
          .set({
            plan: ws.planDowngradeRequest,
            planChanged: null,
            planDowngradeRequest: null,
            subscriptions: ws.planDowngradeRequest === "free" ? null : undefined,
          })
          .where(eq(schema.workspaces.id, ws.id)),
      );
    }
  }
}

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
