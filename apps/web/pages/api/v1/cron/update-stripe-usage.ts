import { stripeEnv } from "@/lib/env";
import { db, isNotNull, schema } from "@/lib/db";
import { getTotalActiveKeys, getTotalVerificationsForWorkspace } from "@/lib/tinybird";
import Stripe from "stripe";
import { NextApiRequest, NextApiResponse } from "next";
import { verifySignature } from "@upstash/qstash/nextjs";

async function handler(_req: NextApiRequest, res: NextApiResponse) {
  if (!stripeEnv) {
    res.status(500).send("STRIPE_SECRET_KEY is missing");
    res.end();
    return;
  }

  const stripe = new Stripe(stripeEnv.STRIPE_SECRET_KEY, {
    apiVersion: "2022-11-15",
  });

  const workspaces = await db.query.workspaces.findMany({
    where: isNotNull(schema.workspaces.stripeSubscriptionId),
  });

  for (const ws of workspaces) {
    console.log("handling workspace: %s", ws.id);
    if (!ws.stripeSubscriptionId) {
      console.error("workspace %s should have a subscriptionId", ws.id);
      continue;
    }
    const subscription = await stripe.subscriptions.retrieve(ws.stripeSubscriptionId);
    if (!subscription) {
      console.error("subscription not found", ws.stripeSubscriptionId);
      continue;
    }
    const start = subscription.current_period_start * 1000;
    const end = subscription.current_period_end * 1000;
    for (const item of subscription.items.data) {
      console.log("handling item %s -> product %s", item.id, item.price.product);
      switch (item.price.product) {
        case stripeEnv.STRIPE_ACTIVE_KEYS_PRODUCT_ID: {
          const usage = await getTotalActiveKeys({ workspaceId: ws.id, start, end });
          const quantity = usage.data.at(0)?.usage;
          if (quantity) {
            await stripe.subscriptionItems.createUsageRecord(item.id, {
              timestamp: Math.floor(Date.now() / 1000),
              action: "set",
              quantity,
            });
            console.log("updated active keys for %s: %d", item.id, quantity);
          }
          break;
        }
        case stripeEnv.STRIPE_KEY_VERIFICATIONS_PRODUCT_ID: {
          const usage = await getTotalVerificationsForWorkspace({
            workspaceId: ws.id,
            start,
            end,
          });
          const quantity = usage.data.at(0)?.usage;
          if (quantity) {
            await stripe.subscriptionItems.createUsageRecord(item.id, {
              timestamp: Math.floor(Date.now() / 1000),
              action: "set",
              quantity,
            });
            console.log("updated verifications for %s: %d", item.id, quantity);
          }
          break;
        }
      }
    }
  }

  res.send("OK");
  res.end();
}

export default process.env.NODE_ENV === "production" ? verifySignature(handler) : handler;
