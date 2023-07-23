import { NextApiRequest, NextApiResponse } from "next";
import Stripe from "stripe";
import { Readable } from "node:stream";
import { db, eq, schema } from "@/lib/db";
import { z } from "zod";
import { stripeEnv } from "@/lib/env";

// Stripe requires the raw body to construct the event.
export const config = {
  api: {
    bodyParser: false,
  },
};

async function buffer(readable: Readable) {
  const chunks = [];
  for await (const chunk of readable) {
    chunks.push(typeof chunk === "string" ? Buffer.from(chunk) : chunk);
  }
  return Buffer.concat(chunks);
}

const _relevantEvents = new Set([
  "customer.subscription.created",
  "customer.subscription.updated",
  "customer.subscription.deleted",
]);

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

    const stripe = new Stripe(stripeEnv.STRIPE_SECRET_KEY, {
      apiVersion: "2022-11-15",
      typescript: true,
    });

    const event = stripe.webhooks.constructEvent(
      (await buffer(req)).toString(),
      signature,
      stripeEnv.STRIPE_WEBHOOK_SECRET,
    );

    switch (event.type) {
      case "customer.subscription.created":
      case "customer.subscription.updated": {
        const newSubscription = event.data.object as Stripe.Subscription;
        await db
          .update(schema.workspaces)
          .set({
            stripeCustomerId: newSubscription.customer.toString(),
            stripeSubscriptionId: newSubscription.id,
            plan: "pro",
          })
          .where(eq(schema.workspaces.stripeCustomerId, newSubscription.customer.toString()));

        break;
      }

      case "customer.subscription.deleted": {
        const subscription = event.data.object as Stripe.Subscription;
        console.log("subscription deleted", subscription);

        const ws = await db.query.workspaces.findFirst({
          where: eq(schema.workspaces.stripeCustomerId, subscription.customer.toString()),
        });
        if (!ws) {
          throw new Error("workspace does not exist");
        }
        await db
          .update(schema.workspaces)
          .set({
            stripeCustomerId: subscription.customer.toString(),
            stripeSubscriptionId: null,
            plan: "free",
          })
          .where(eq(schema.workspaces.id, ws.id));

        break;
      }
      default:
        console.error("Incoming stripe event, that should not be received", event.type);
        break;
    }
    res.send("OK");
  } catch (e) {
    const err = e as Error;
    console.error(err.message);
    res.status(500).send(err.message);
    return;
  } finally {
    res.end();
  }
}
