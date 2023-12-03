import { db, eq, schema } from "@/lib/db";
import { env, stripeEnv } from "@/lib/env";
import {
  getActiveKeysPerHourForAllWorkspaces,
  getVerificationsPerHourForAllWorkspaces,
} from "@/lib/tinybird";
import { verifySignature } from "@upstash/qstash/nextjs";
import { NextApiRequest, NextApiResponse } from "next";
import Stripe from "stripe";
export const config = {
  maxDuration: 300,
  runtime: "nodejs",
};

async function handler(_req: NextApiRequest, res: NextApiResponse) {
  try {
    const e = stripeEnv();
    if (!e) {
      throw new Error("STRIPE_SECRET_KEY is missing");
    }

    const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
      apiVersion: "2022-11-15",
    });

    const allWorkspaces = await db.query.workspaces.findMany();

    console.log("found %d workspaces", allWorkspaces.length);

    const globalActiveKeys = (await getActiveKeysPerHourForAllWorkspaces({})).data;
    const globalVerifications = (await getVerificationsPerHourForAllWorkspaces({})).data;
    while (allWorkspaces.length > 0) {
      const workspaces = allWorkspaces.splice(0, 20);

      await Promise.all(
        workspaces.map(async (ws) => {
          let [start, end] = getMonthStartAndEnd();
          let subscription: Stripe.Response<Stripe.Subscription> | null = null;

          if (ws.stripeSubscriptionId) {
            console.log("fetching subscription %s", ws.stripeSubscriptionId);
            subscription = await stripe.subscriptions.retrieve(ws.stripeSubscriptionId);
            if (!subscription) {
              throw new Error(`subscription not found: ${ws.stripeSubscriptionId}`);
            }
            start = subscription.current_period_start * 1000;
            end = subscription.current_period_end * 1000;
          }
          let activeKeys = 0;
          for (const d of globalActiveKeys) {
            if (
              d.workspaceId === ws.id &&
              d.time > start &&
              d.time <= end &&
              d.usage > activeKeys
            ) {
              activeKeys = d.usage;
            }
          }
          console.log({ activeKeys });

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

          await db
            .update(schema.workspaces)
            .set({
              usageActiveKeys: activeKeys,
              usageVerifications: verifications,
              lastUsageUpdate: new Date(),
            })
            .where(eq(schema.workspaces.id, ws.id));

          if (subscription) {
            for (const item of subscription.items.data) {
              console.log("handling item %s -> product %s", item.id, item.price.product);
              switch (item.price.product) {
                case e.STRIPE_PRODUCT_ID_ACTIVE_KEYS: {
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
                case e.STRIPE_PRODUCT_ID_KEY_VERIFICATIONS: {
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
    }

    // report success
    if (env().HEARTBEAT_UPDATE_USAGE_URL) {
      await fetch(env().HEARTBEAT_UPDATE_USAGE_URL!);
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
