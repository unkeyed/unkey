import { Readable } from "node:stream";
import { QUOTA } from "@/lib/constants/quotas";
import { db, eq, schema } from "@/lib/db";
import { env, stripeEnv } from "@/lib/env";
import { clerkClient } from "@clerk/nextjs";
import { Loops } from "@unkey/loops";
import { NextApiRequest, NextApiResponse } from "next";
import Stripe from "stripe";
import { z } from "zod";

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
        const sub = event.data.object as Stripe.Subscription;
        await db
          .update(schema.workspaces)
          .set({
            stripeCustomerId: sub.customer.toString(),
            stripeSubscriptionId: sub.id,
            plan: "pro",
            billingPeriodStart: new Date(sub.current_period_start * 1000),
            billingPeriodEnd: new Date(sub.current_period_end * 1000),
            trialEnds: sub.trial_end ? new Date(sub.trial_end * 1000) : null,
            maxActiveKeys: QUOTA.pro.maxActiveKeys,
            maxVerifications: QUOTA.pro.maxVerifications,
          })
          .where(eq(schema.workspaces.stripeCustomerId, sub.customer.toString()));

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
            billingPeriodStart: null,
            billingPeriodEnd: null,
          })
          .where(eq(schema.workspaces.id, ws.id));

        break;
      }
      case "customer.subscription.trial_will_end": {
        const subscription = event.data.object as Stripe.Subscription;
        console.log("subscription will end", subscription);
        if (!env.LOOPS_API_KEY) {
          // no need to fetch everything if we don't use it
          break;
        }
        const ws = await db.query.workspaces.findFirst({
          where: eq(schema.workspaces.stripeCustomerId, subscription.customer.toString()),
        });
        if (!ws) {
          throw new Error("workspace does not exist");
        }

        const userIds: string[] = [];
        if (ws.tenantId.startsWith("org_")) {
          const members = await clerkClient.organizations.getOrganizationMembershipList({
            organizationId: ws.tenantId,
          });
          for (const m of members) {
            userIds.push(m.publicUserData!.userId);
          }
        } else {
          userIds.push(ws.tenantId);
        }
        const loops = await new Loops({ apiKey: env.LOOPS_API_KEY });
        for await (const userId of userIds) {
          const user = await clerkClient.users.getUser(userId);
          const email = user.emailAddresses.at(0)?.emailAddress;
          if (!email) {
            console.warn("user doesn't have an email: %s", user.id);
            continue;
          }
          await loops.sendTrialEnds({
            email,
            name: user.firstName ?? user.username ?? "",
            date: new Date(subscription.trial_end! * 1000),
          });
        }

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
