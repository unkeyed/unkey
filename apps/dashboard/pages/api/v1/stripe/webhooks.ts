import type { Readable } from "node:stream";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import type { NextApiRequest, NextApiResponse } from "next";
import Stripe from "stripe";
import { z } from "zod";
// Stripe requires the raw body to construct the event.
export const config = {
  api: {
    bodyParser: false,
  },
  runtime: "nodejs",
};

async function buffer(readable: Readable) {
  const chunks = [];
  for await (const chunk of readable) {
    chunks.push(typeof chunk === "string" ? Buffer.from(chunk) : chunk);
  }
  return Buffer.concat(chunks);
}

const requestValidation = z.object({
  method: z.literal("POST"),
  headers: z.object({
    "stripe-signature": z.string(),
  }),
});

export default async function webhookHandler(req: NextApiRequest, res: NextApiResponse) {
  try {
    const {
      headers: { "stripe-signature": signature },
    } = requestValidation.parse(req);

    if (!stripeEnv) {
      throw new Error("stripe env variables are not set up");
    }

    const stripe = new Stripe(stripeEnv()!.STRIPE_SECRET_KEY, {
      apiVersion: "2023-10-16",
      typescript: true,
    });

    const event = stripe.webhooks.constructEvent(
      (await buffer(req)).toString(),
      signature,
      stripeEnv()!.STRIPE_WEBHOOK_SECRET,
    );

    switch (event.type) {
      case "customer.subscription.created":
      case "customer.subscription.updated": {
        const sub = event.data.object as Stripe.Subscription;

        // https://docs.stripe.com/billing/subscriptions/webhooks#state-changes
        switch (sub.status) {
          case "trialing":
          case "active": {
            const product = await stripe.products.retrieve(
              sub.items.data[0].price.product.toString(),
            );

            const workspace = await db.query.workspaces.findFirst({
              where: (table, { eq }) => eq(table.stripeCustomerId, sub.customer.toString()),
            });
            if (!workspace) {
              throw new Error(`No workspace exists for ${sub.customer}`);
            }

            await db.transaction(async (tx) => {
              await tx
                .update(schema.workspaces)
                .set({
                  tier: product.name,
                  stripeSubscriptionId: sub.id,
                })
                .where(eq(schema.workspaces.id, workspace.id));
              await tx
                .update(schema.quotas)
                .set({
                  requestsPerMonth: Number.parseInt(product.metadata.quota_requests_per_month),
                  logsRetentionDays: Number.parseInt(product.metadata.quota_logs_retention_days),
                  auditLogsRetentionDays: Number.parseInt(
                    product.metadata.quota_audit_logs_retention_days,
                  ),
                  team: true,
                })
                .where(eq(schema.quotas.workspaceId, workspace.id));
            });
            break;
          }
          case "canceled":
          case "paused": {
            const workspace = await db.query.workspaces.findFirst({
              where: (table, { eq }) => eq(table.stripeCustomerId, sub.customer.toString()),
            });
            if (!workspace) {
              throw new Error(`No workspace exists for ${sub.customer}`);
            }

            await db.transaction(async (tx) => {
              await tx
                .update(schema.workspaces)
                .set({
                  tier: "Free",
                  stripeSubscriptionId: sub.id,
                })
                .where(eq(schema.workspaces.id, workspace.id));
              await tx
                .update(schema.quotas)
                .set({
                  requestsPerMonth: 250000,
                  logsRetentionDays: 7,
                  auditLogsRetentionDays: 30,
                  team: false,
                })
                .where(eq(schema.quotas.workspaceId, workspace.id));
            });
            break;
          }

          //  case "incomplete":
          //  case "incomplete_expired":
          //  case "past_due":
          //  case "unpaid":
        }
        break;
      }
      case "customer.subscription.trial_will_end":
        break;
      case "customer.subscription.deleted": {
        const sub = event.data.object as Stripe.Subscription;

        const workspace = await db.query.workspaces.findFirst({
          where: (table, { eq }) => eq(table.stripeCustomerId, sub.customer.toString()),
        });
        if (!workspace) {
          throw new Error(`No workspace exists for ${sub.customer}`);
        }

        await db.transaction(async (tx) => {
          await tx
            .update(schema.workspaces)
            .set({
              tier: "Free",
              stripeSubscriptionId: null,
            })
            .where(eq(schema.workspaces.id, workspace.id));
          await tx
            .update(schema.quotas)
            .set({
              requestsPerMonth: 250000,
              logsRetentionDays: 7,
              auditLogsRetentionDays: 30,
              team: false,
            })
            .where(eq(schema.quotas.workspaceId, workspace.id));
        });
        break;
      }

      default:
        console.error("Incoming stripe event, that should not be received", event.type);
        break;
    }
    res.send("OK");
  } catch (e) {
    const err = e as Error;
    console.error(err);
    res.status(500).send(err.message);
    return;
  } finally {
    res.end();
  }
}
