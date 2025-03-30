import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { freeTierQuotas } from "@/lib/quotas";
import Stripe from "stripe";

export const runtime = "nodejs";

export const POST = async (req: Request): Promise<Response> => {
  const signature = req.headers.get("stripe-signature");
  if (!signature) {
    throw new Error("Signature missing");
  }

  const e = stripeEnv();

  if (!e) {
    throw new Error("stripe env variables are not set up");
  }

  const stripe = new Stripe(stripeEnv()!.STRIPE_SECRET_KEY, {
    apiVersion: "2023-10-16",
    typescript: true,
  });

  const event = stripe.webhooks.constructEvent(
    await req.text(),
    signature,
    e.STRIPE_WEBHOOK_SECRET,
  );

  switch (event.type) {
    case "customer.subscription.deleted": {
      const sub = event.data.object as Stripe.Subscription;

      const ws = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.stripeSubscriptionId, sub.id), isNull(table.deletedAtM)),
      });
      if (!ws) {
        return new Response("workspace does not exist", { status: 500 });
      }
      await db
        .update(schema.workspaces)
        .set({
          stripeSubscriptionId: null,
        })
        .where(eq(schema.workspaces.id, ws.id));

      await db
        .insert(schema.quotas)
        .values({
          workspaceId: ws.id,
          ...freeTierQuotas,
        })
        .onDuplicateKeyUpdate({
          set: freeTierQuotas,
        });

      await insertAuditLogs(db, {
        workspaceId: ws.id,
        actor: {
          type: "system",
          id: "stripe",
        },
        event: "workspace.update",
        description: "Cancelled subscription.",
        resources: [],
        context: {
          location: "",
          userAgent: undefined,
        },
      });
      break;
    }
    case "customer.subscription.created": {
      const sub = event.data.object as Stripe.Subscription;
      
      // Only handle trial subscriptions
      if (!sub.trial_end) {
      return new Response("OK");
      }

      // Get product and price information
      const price = await stripe.prices.retrieve(sub.items.data[0].price.id);
      const product = await stripe.products.retrieve(price.product as string);
      const customer = await stripe.customers.retrieve(sub.customer as string) as Stripe.Customer;

      const formattedPrice = new Intl.NumberFormat("en-US", {
        style: "currency",
        currency: "USD",
      }).format(price.unit_amount! / 100);


      await alertSlack(product.name, formattedPrice, customer.email!, customer.name!);
      break;
    }

    default:
      console.warn("Incoming stripe event, that should not be received", event.type);
      break;
  }
  return new Response("OK");
};


async function alertSlack(product: string, price: string, email: string, name?: string): Promise<void> {
  const url = process.env.SLACK_WEBHOOK_CUSTOMERS;
  if (!url) {
    return;
  }
  

  await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      "blocks": [
        {
          "type": "section",
          "text": {
            "type": "mrkdwn",
            "text": `:bugeyes: New customer ${name} signed up for a two week trial`
          }
        },
    	{
    		"type": "section",
    		"text": {
    			"type": "mrkdwn",
    			"text": `A new trial for the ${product} tier has started at a price of ${price} by ${email} :moneybag: `
    		}
    	},
    ]
    }),
  }).catch((err: Error) => {
    console.error(err);
  });
}