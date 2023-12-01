import { Readable } from "node:stream";
import { QUOTA } from "@/lib/constants/quotas";
import { db, eq, schema } from "@/lib/db";
import { env, stripeEnv } from "@/lib/env";
import { clerkClient } from "@clerk/nextjs";
import PaymentIssue from "@unkey/resend/emails/payment_issue";
import SubscriptionEnded from "@unkey/resend/emails/subscription_ended";
import TrialEndsIn3Days from "@unkey/resend/emails/trial_ends_in_3_days";
import { NextApiRequest, NextApiResponse } from "next";
import { Resend } from "resend";
import Stripe from "stripe";
import { z } from "zod";
// Stripe requires the raw body to construct the event.
export const config = {
  api: {
    bodyParser: false,
  },
  runtime: "nodejs",
};
const domain = "updates.unkey.dev";
const replyTo = "support@unkey.dev";

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
const email = env().RESEND_API_KEY ? new Resend(env().RESEND_API_KEY!) : null;

export default async function webhookHandler(req: NextApiRequest, res: NextApiResponse) {
  try {
    const {
      headers: { "stripe-signature": signature },
    } = requestValidation.parse(req);

    if (!stripeEnv) {
      throw new Error("stripe env variables are not set up");
    }

    const stripe = new Stripe(stripeEnv()!.STRIPE_SECRET_KEY, {
      apiVersion: "2022-11-15",
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
        await db
          .update(schema.workspaces)
          .set({
            stripeCustomerId: sub.customer.toString(),
            stripeSubscriptionId: sub.id,
            plan: "pro",
            billingPeriodStart: new Date(sub.current_period_start * 1000),
            billingPeriodEnd: new Date(sub.current_period_end * 1000),
            trialEnds: null,
            maxActiveKeys: QUOTA.pro.maxActiveKeys,
            maxVerifications: QUOTA.pro.maxVerifications,
          })
          .where(eq(schema.workspaces.stripeCustomerId, sub.customer.toString()));

        break;
      }

      case "customer.subscription.deleted": {
        const subscription = event.data.object as Stripe.Subscription;
        console.log("subscription deleted", subscription.id);
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

        if (email) {
          const users = await getUsers(ws.tenantId);
          for await (const user of users) {
            await email.contacts.remove({
              audience_id: env().RESEND_AUDIENCE_ID!,
              id: user.email,
            });
            await email.emails.send({
              to: user.email,
              from: `james@${domain}`,
              reply_to: replyTo,
              subject: "Your Unkey subscription has ended",
              react: SubscriptionEnded({ username: user.name }),
            });
          }
        }
        break;
      }
      case "customer.subscription.trial_will_end": {
        const subscription = event.data.object as Stripe.Subscription;
        console.log("subscription will end", subscription.id);
        if (!email) {
          // no need to fetch everything if we don't use it
          break;
        }
        const ws = await db.query.workspaces.findFirst({
          where: eq(schema.workspaces.stripeCustomerId, subscription.customer.toString()),
        });
        if (!ws) {
          throw new Error("workspace does not exist");
        }

        const users = await getUsers(ws.tenantId);
        for await (const user of users) {
          await email.emails.send({
            to: user.email,
            from: `james@${domain}`,
            reply_to: replyTo,
            subject: "Your Unkey trial ends in 3 days",
            react: TrialEndsIn3Days({
              workspaceName: ws.name,
              username: user.name,
              endDate: new Date(subscription.trial_end! * 1000).toString(),
            }),
          });
        }

        break;
      }
      case "invoice.payment_failed": {
        const invoice = event.data.object as Stripe.Invoice;
        console.log("invoice failed", invoice.id);
        if (!email) {
          break;
        }
        const ws = await db.query.workspaces.findFirst({
          where: eq(schema.workspaces.stripeCustomerId, invoice.customer!.toString()),
        });
        if (!ws) {
          throw new Error("workspace does not exist");
        }
        const users = await getUsers(ws.tenantId);
        const date = invoice.effective_at ? new Date(invoice.effective_at * 1000) : new Date();
        for await (const user of users) {
          await email.emails.send({
            to: user.email,
            from: `james@${domain}`,
            reply_to: replyTo,
            subject: "There was an issue with your payment",
            react: PaymentIssue({
              username: user.name,
              date: date.toString(),
            }),
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

async function getUsers(tenantId: string): Promise<{ id: string; email: string; name: string }[]> {
  const userIds: string[] = [];
  if (tenantId.startsWith("org_")) {
    const members = await clerkClient.organizations.getOrganizationMembershipList({
      organizationId: tenantId,
    });
    for (const m of members) {
      userIds.push(m.publicUserData!.userId);
    }
  } else {
    userIds.push(tenantId);
  }

  return await Promise.all(
    userIds.map(async (userId) => {
      const user = await clerkClient.users.getUser(userId);
      const email = user.emailAddresses.at(0)?.emailAddress;
      if (!email) {
        throw new Error(`user ${user.id} does not have an email`);
      }
      return {
        id: user.id,
        name: user.firstName ?? user.username ?? "there",
        email,
      };
    }),
  );
}
