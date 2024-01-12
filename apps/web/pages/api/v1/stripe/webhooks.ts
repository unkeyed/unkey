import { Readable } from "node:stream";
import { db } from "@/lib/db";
import { env, stripeEnv } from "@/lib/env";
import { clerkClient } from "@clerk/nextjs";
import { Resend } from "@unkey/resend";
import { NextApiRequest, NextApiResponse } from "next";
import Stripe from "stripe";
import { z } from "zod";
// Stripe requires the raw body to construct the event.
export const config = {
  api: {
    bodyParser: false,
  },
  runtime: "nodejs",
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
    const resend = env().RESEND_API_KEY ? new Resend({ apiKey: env().RESEND_API_KEY! }) : null;
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
      case "invoice.payment_failed": {
        const invoice = event.data.object as Stripe.Invoice;
        console.log("invoice failed", invoice.id);
        if (!resend) {
          break;
        }
        const ws = await db.query.workspaces.findFirst({
          where: (table, { and, eq, isNull }) =>
            and(eq(table.stripeCustomerId, invoice.customer!.toString()), isNull(table.deletedAt)),
        });
        if (!ws) {
          throw new Error("workspace does not exist");
        }
        const users = await getUsers(ws.tenantId);
        const date = invoice.effective_at ? new Date(invoice.effective_at * 1000) : new Date();
        for await (const user of users) {
          await resend.sendPaymentIssue({
            email: user.email,
            name: user.name,
            date: date,
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
