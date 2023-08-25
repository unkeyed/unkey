import { env, stripeEnv } from "@/lib/env";
import { Workspace, db, schema } from "@/lib/db";
import { getTotalActiveKeys, getTotalVerificationsForWorkspace } from "@/lib/tinybird";
import Stripe from "stripe";
import { NextApiRequest, NextApiResponse } from "next";
import { verifySignature } from "@upstash/qstash/nextjs";

async function handler(_req: NextApiRequest, res: NextApiResponse) {
  try {
    if (!stripeEnv) {
      res.status(500).send("STRIPE_SECRET_KEY is missing");
      res.end();
      return;
    }

    const stripe = new Stripe(stripeEnv.STRIPE_SECRET_KEY, {
      apiVersion: "2022-11-15",
    });

    const workspaceBatchSize = 50;
    let workspaceOffset = 0;

    let workspaces: Workspace[] = [];
    do {
      workspaces = await db.query.workspaces.findMany({
        limit: workspaceBatchSize,
        offset: workspaceOffset,
        orderBy: schema.workspaces.id,
      });
      workspaceOffset += workspaces.length;

      await Promise.all(
        workspaces.map(async (ws) => {
          let [start, end] = getMonthStartAndEnd();
          let subscription: Stripe.Response<Stripe.Subscription> | null = null;

          if (ws.stripeSubscriptionId) {
            subscription = await stripe.subscriptions.retrieve(ws.stripeSubscriptionId);
            if (!subscription) {
              throw new Error(`subscription not found: ${ws.stripeSubscriptionId}`);
            }
            start = subscription.current_period_start * 1000;
            end = subscription.current_period_end * 1000;
          }

          const activeKeys = await getTotalActiveKeys({ workspaceId: ws.id, start, end }).then(
            (res) => res.data.at(0)?.usage,
          );
          const verifications = await getTotalVerificationsForWorkspace({
            workspaceId: ws.id,
            start,
            end,
          }).then((res) => res.data.at(0)?.usage);

          console.log(`updating workspace ${ws.id}`, {
            start,
            end,
            activeKeys,
            verifications,
            subscriptionId: subscription?.id,
          });

          await db.update(schema.workspaces).set({
            usageActiveKeys: activeKeys ?? 0,
            usageVerifications: verifications ?? 0,
            lastUsageUpdate: new Date(),
          });
          if (subscription) {
            for (const item of subscription.items.data) {
              console.log("handling item %s -> product %s", item.id, item.price.product);
              switch (item.price.product) {
                case stripeEnv!.STRIPE_ACTIVE_KEYS_PRODUCT_ID: {
                  if (activeKeys) {
                    await stripe.subscriptionItems.createUsageRecord(item.id, {
                      timestamp: Math.floor(Date.now() / 1000),
                      action: "set",
                      quantity: activeKeys,
                    });
                    console.log("updated active keys for %s: %d", item.id, activeKeys);
                  }
                  break;
                }
                case stripeEnv!.STRIPE_KEY_VERIFICATIONS_PRODUCT_ID: {
                  if (verifications) {
                    await stripe.subscriptionItems.createUsageRecord(item.id, {
                      timestamp: Math.floor(Date.now() / 1000),
                      action: "set",
                      quantity: verifications,
                    });
                    console.log("updated verifications for %s: %d", item.id, verifications);
                  }
                  break;
                }
              }
            }
          }
        }),
      );
    } while (workspaces.length === workspaceBatchSize);

    // report success
    if (env.UPTIME_CRON_URL_COLLECT_BILLING) {
      await fetch(env.UPTIME_CRON_URL_COLLECT_BILLING);
    }

    res.send("OK");
  } catch (err) {
    res.status(500);
    if (err instanceof Error) {
      console.error(err.message);
      res.send(err.message);
    } else {
      res.send("Shit broke");
    }
  } finally {
    res.end();
  }
}

export default process.env.NODE_ENV === "production" ? verifySignature(handler) : handler;

/**
 *
 * return utc start and end time of the month as unix milliseconds timestamps
 */
function getMonthStartAndEnd(): [number, number] {
  const t = new Date();
  t.setUTCDate(0);
  t.setUTCHours(0, 0, 0, 0);

  const start = t.getTime() + 1;

  t.setUTCMonth(t.getUTCMonth() + 1);
  const end = t.getTime();

  return [start, end];
}
