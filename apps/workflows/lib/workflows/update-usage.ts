import { connectDatabase, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { inngest } from "@/lib/inngest";
import { Tinybird } from "@/lib/tinybird";
import Stripe from "stripe";
export const config = {
  maxDuration: 300,
};

export const updateUsage = inngest.createFunction(
  { id: "update.usage" },
  { cron: "0 0 * * *" },
  async ({ event, step, logger }) => {
    const db = connectDatabase();
    const stripe = new Stripe(env().STRIPE_SECRET_KEY, {
      apiVersion: "2022-11-15",
      typescript: true,
    });
    const tinybird = new Tinybird(env().TINYBIRD_TOKEN);

    const workspaces = await step.run("get-paid-workspaces", async () =>
      db.query.workspaces.findMany({
        where: (table, { isNotNull }) => isNotNull(table.stripeSubscriptionId),
      }),
    );

    logger.info("found workspaces: ", workspaces.length);

    const globalActiveKeys = await step.run("get-active-keys", async () =>
      tinybird.activeKeys({}).then((res) => res.data),
    );
    const globalVerifications = await step.run("get-verifications", async () =>
      tinybird.verifications({}).then((res) => res.data),
    );

    await Promise.all(
      workspaces.map(async (ws) => {
        logger.info("workspace", ws.id);

        const subscription = await step.run(
          `get-stripe-subscription-${ws.stripeSubscriptionId}`,
          async () => {
            const s = await stripe.subscriptions.retrieve(ws.stripeSubscriptionId!);
            if (!s) {
              throw new Error(`subscription not found: ${ws.stripeSubscriptionId}`);
            }
            return s;
          },
        );

        const start = subscription.current_period_start * 1000;
        const end = subscription.current_period_end * 1000;

        let activeKeys = 0;
        for (const d of globalActiveKeys) {
          if (d.workspaceId === ws.id && d.time > start && d.time <= end && d.usage > activeKeys) {
            activeKeys = d.usage;
          }
        }

        let verifications = 0;
        for (const d of globalVerifications) {
          if (d.workspaceId === ws.id && d.time > start && d.time <= end) {
            verifications += d.verifications;
          }
        }
        if (verifications > 0 || activeKeys > 0) {
          console.log(
            "%s did %d verifications with %d keys between %s and %s",
            ws.id,
            verifications,
            activeKeys,
            new Date(start).toDateString(),
            new Date(end).toDateString(),
          );
        }

        await step.run(`update-database-${ws.id}`, async () =>
          db
            .update(schema.workspaces)
            .set({
              usageActiveKeys: activeKeys,
              usageVerifications: verifications,
              lastUsageUpdate: new Date(),
            })
            .where(eq(schema.workspaces.id, ws.id)),
        );

        await step.run(`update-usage-subscription-${ws.id}`, async () => {
          for (const item of subscription!.items.data) {
            logger.info(`handling item ${item.id} -> product ${item.price.product}`);
            switch (item.price.product) {
              case env().STRIPE_ACTIVE_KEYS_PRODUCT_ID: {
                if (activeKeys) {
                  await stripe.subscriptionItems.createUsageRecord(item.id, {
                    timestamp: Math.floor(Date.now() / 1000),
                    action: "set",
                    quantity: activeKeys,
                  });
                  logger.info(`updated active keys for ${item.id}: ${activeKeys}`);
                }
                break;
              }
              case env().STRIPE_KEY_VERIFICATIONS_PRODUCT_ID: {
                if (verifications) {
                  await stripe.subscriptionItems.createUsageRecord(item.id, {
                    timestamp: Math.floor(Date.now() / 1000),
                    action: "set",
                    quantity: verifications,
                  });
                  logger.info(`updated verifications for ${item.id}: ${verifications}`);
                }
                break;
              }
            }
          }
        });
      }),
    );

    // report success
    await fetch(env().HEARTBEAT_UPDATE_USAGE_URL!);

    return {
      event,
      body: "done",
    };
  },
);
