import { stripeEnv, cronEnv } from "@/lib/env";
import { db, isNotNull, schema } from "@/lib/db";
import { getTotalActiveKeys, getTotalVerificationsForWorkspace } from "@/lib/tinybird";
import Stripe from "stripe";
import { NextApiRequest, NextApiResponse } from "next";

export default async function (req: NextApiRequest, res: NextApiResponse) {
  if (!cronEnv || req.query.key !== cronEnv.CRON_STRIPE_AUTH_KEY) {
    res.status(404).end();
    return;
  }
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
  await Promise.all(
    workspaces.map(async (ws) => {
      const subscription = await stripe.subscriptions.retrieve(ws.stripeSubscriptionId!);
      if (!subscription) {
        console.error("subscription not found", ws.stripeSubscriptionId);
        return;
      }
      const start = subscription.current_period_start * 1000;
      const end = subscription.current_period_end * 1000;
      for (const item of subscription.items.data) {
        switch (item.price.product) {
          case stripeEnv?.STRIPE_ACTIVE_KEYS_PRODUCT_ID: {
            const usage = await getTotalActiveKeys({ workspaceId: ws.id, start, end });
            const quantity = usage.data.at(0)?.usage;
            if (quantity) {
              await stripe.subscriptionItems.createUsageRecord(item.id, {
                action: "set",
                quantity: 0,
              });
            }
            console.log("updated quantity");
            break;
          }
          case stripeEnv?.STRIPE_KEY_VERIFICATIONS_PRODUCT_ID: {
            const usage = await getTotalVerificationsForWorkspace({
              workspaceId: ws.id,
              start,
              end,
            });
            const quantity = usage.data.at(0)?.usage;
            if (quantity) {
              await stripe.subscriptionItems.createUsageRecord(item.id, {
                action: "set",
                quantity: 0,
              });
            }
            break;
          }
        }
      }
    }),
  );

  res.send("OK");
  res.end();
}
