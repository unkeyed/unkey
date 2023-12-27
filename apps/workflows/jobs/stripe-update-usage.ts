import { connectDatabase } from "@/lib/db";
import { env } from "@/lib/env";
import { Tinybird } from "@/lib/tinybird";
import { client } from "@/trigger";
import { cronTrigger } from "@trigger.dev/sdk";
import { Stripe } from "stripe";

client.defineJob({
  id: "stripe.updateUsage",
  name: "Update usage in stripe",
  version: "0.0.1",
  trigger: cronTrigger({
    cron: "1 * * * *", // every hour at 1 minute past the hour
  }),

  run: async (_payload, io, _ctx) => {
    const stripe = new Stripe(env().STRIPE_SECRET_KEY, {
      apiVersion: "2022-11-15",
      typescript: true,
    });
    const db = connectDatabase();
    const tb = new Tinybird(env().TINYBIRD_TOKEN);

    const subscribedWorkspaces = await io.runTask("list subscribed workspaces", () =>
      db.query.workspaces.findMany({
        where: (table, { isNotNull }) => isNotNull(table.stripeSubscriptionId),
        columns: {
          id: true,
          stripeCustomerId: true,
          stripeSubscriptionId: true,
        },
      }),
    );
    io.logger.info(`found ${subscribedWorkspaces.length} subscribed workspaces`);

    const summary: Record<
      string,
      {
        activeKeys: number;
        verifications: number;
      }
    > = {};

    for (const ws of subscribedWorkspaces) {
      summary[ws.id] = { activeKeys: 0, verifications: 0 };
      const subscriptionId = ws.stripeSubscriptionId!;
      const subscription = await io.runTask(`fetch subscription ${subscriptionId}`, async () => {
        const s = await stripe.subscriptions.retrieve(subscriptionId);
        return {
          id: s.id,
          currentPeriodStart: toStartOfHour(s.current_period_start * 1000),
          currentPeriodEnd: toStartOfHour(s.current_period_end * 1000),
          items: s.items.data.map((i) => ({
            id: i.id,
            productId: i.price.product as string,
            priceId: i.price.id,
          })),
        };
      });
      for (const item of subscription.items) {
        switch (item.productId) {
          case env().STRIPE_PRODUCT_ID_ACTIVE_KEYS: {
            const activeKeys = await io.runTask(`fetch active keys for ${ws.id}`, async () =>
              tb
                .activeKeys({
                  workspaceId: ws.id,
                  start: subscription.currentPeriodStart,
                  end: subscription.currentPeriodEnd,
                })
                .then((res) => res.data.at(0)?.activeKeys ?? 0),
            );
            summary[ws.id].activeKeys = activeKeys;
            if (activeKeys > 0) {
              await io.runTask(`update active keys usage for ${subscriptionId}`, async () => {
                await stripe.subscriptionItems.createUsageRecord(item.id, {
                  timestamp: Math.floor(Date.now() / 1000),
                  action: "set",
                  quantity: activeKeys,
                });
              });
              await io.logger.info(`updated key usage for ${subscriptionId}`, { activeKeys });
            }
            break;
          }

          case env().STRIPE_PRODUCT_ID_KEY_VERIFICATIONS: {
            const verifications = await io.runTask(`fetch verifications for ${ws.id}`, async () =>
              tb
                .verifications({
                  workspaceId: ws.id,
                  start: subscription.currentPeriodStart,
                  end: subscription.currentPeriodEnd,
                })
                .then((res) => res.data.at(0)?.verifications ?? 0),
            );
            summary[ws.id].verifications = verifications;
            if (verifications > 0) {
              await io.runTask(`update verification usage for ${subscriptionId}`, async () => {
                await stripe.subscriptionItems.createUsageRecord(item.id, {
                  timestamp: Math.floor(Date.now() / 1000),
                  action: "set",
                  quantity: verifications,
                });
              });
              await io.logger.info(`updated verification usage for ${subscriptionId}`, {
                verifications,
              });
            }
            break;
          }
        }
      }
    }
    return { summary };
  },
});

/**
 * Returns the start of an hour for the given millisecond-timestamp.
 */
function toStartOfHour(t: number): number {
  const d = new Date(t);
  d.setUTCMinutes(0, 0, 0);
  return d.getTime();
}
