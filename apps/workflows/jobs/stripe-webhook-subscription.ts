import { connectDatabase, eq, schema } from "@/lib/db";
import { client } from "@/trigger";
import { Stripe } from "@trigger.dev/stripe";

const stripe = new Stripe({
  id: "stripe",
  apiKey: process.env.STRIPE_SECRET_KEY!,
});

client.defineJob({
  id: "stripe.subscription",
  name: "Stripe Subscription Updates",
  version: "0.0.1",
  enabled: true,
  trigger: stripe.onCustomerSubscription({
    events: [
      "customer.subscription.created",
      "customer.subscription.updated",
      "customer.subscription.deleted",
    ],
  }),
  run: async (subscription, io, ctx) => {
    const db = connectDatabase();

    const stripeCustomerId = subscription.customer.toString();

    switch (ctx.event.name) {
      case "customer.subscription.created":
      case "customer.subscription.updated":
        await io.runTask(`update database entry for customer ${stripeCustomerId}`, async () => {
          await db
            .update(schema.workspaces)
            .set({
              stripeCustomerId,
              stripeSubscriptionId: subscription.id,
              plan: "pro",
              trialEnds: null,
            })
            .where(eq(schema.workspaces.stripeCustomerId, stripeCustomerId));
        });

      case "customer.subscription.deleted":
        await io.runTask(`update database entry for customer ${stripeCustomerId}`, async () => {
          await db
            .update(schema.workspaces)
            .set({
              stripeSubscriptionId: null,
              plan: "free",
            })
            .where(eq(schema.workspaces.stripeCustomerId, stripeCustomerId));
        });
    }

    return {
      subscriptionId: subscription.id,
    };
  },
});
