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
    throw new Error(
      "Stripe environment configuration is missing. Check that STRIPE_SECRET_KEY and other required Stripe environment variables are properly set.",
    );
  }

  const stripeSecretKey = stripeEnv()?.STRIPE_SECRET_KEY;
  if (!stripeSecretKey) {
    throw new Error(
      "STRIPE_SECRET_KEY environment variable is not set. This is required for Stripe API operations.",
    );
  }

  const stripe = new Stripe(stripeSecretKey, {
    apiVersion: "2023-10-16",
    typescript: true,
  });

  const event = stripe.webhooks.constructEvent(
    await req.text(),
    signature,
    e.STRIPE_WEBHOOK_SECRET,
  );
  switch (event.type) {
    case "customer.subscription.updated": {
      try {
        const sub = event.data.object as Stripe.Subscription;

        const ws = await db.query.workspaces.findFirst({
          where: (table, { and, eq, isNull }) =>
            and(eq(table.stripeSubscriptionId, sub.id), isNull(table.deletedAtM)),
        });
        if (!ws) {
          console.error("Workspace not found for subscription:", sub.id);
          return new Response("OK", { status: 200 });
        }

        // Get the current subscription item and product
        if (!sub.items?.data?.[0]?.price?.id || !sub.customer) {
          return new Response("OK");
        }

        const [price, customer] = await Promise.all([
          stripe.prices.retrieve(sub.items.data[0].price.id),
          stripe.customers.retrieve(
            typeof sub.customer === "string" ? sub.customer : sub.customer.id,
          ),
        ]);

        if (!price.product || price.unit_amount === null || price.unit_amount === undefined) {
          return new Response("OK");
        }

        const product = await stripe.products.retrieve(
          typeof price.product === "string" ? price.product : price.product.id,
        );
        // The subscrtiption is cancelling prior to actually be cancelled.
        if (sub.cancel_at) {
          if (customer && !customer.deleted && customer.email) {
            const formattedPrice = new Intl.NumberFormat("en-US", {
              style: "currency",
              currency: "USD",
            }).format(price.unit_amount / 100);
            await alertIsCancellingSubscription(
              product.name,
              formattedPrice,
              customer.email,
              customer.name || "Unknown",
            );

            return new Response("OK");
          }
        }
        await db
          .update(schema.workspaces)
          .set({
            tier: product.name,
          })
          .where(eq(schema.workspaces.id, ws.id));

        const requiredMetadata = [
          "quota_requests_per_month",
          "quota_logs_retention_days",
          "quota_audit_logs_retention_days",
        ];

        for (const field of requiredMetadata) {
          if (!product.metadata[field]) {
            console.error(`Missing required metadata field: ${field} for product: ${product.id}`);
            return new Response("OK", { status: 200 });
          }
        }

        // Parse and validate quotas
        const requestsPerMonth = Number.parseInt(product.metadata.quota_requests_per_month);
        const logsRetentionDays = Number.parseInt(product.metadata.quota_logs_retention_days);
        const auditLogsRetentionDays = Number.parseInt(
          product.metadata.quota_audit_logs_retention_days,
        );

        if (
          Number.isNaN(requestsPerMonth) ||
          Number.isNaN(logsRetentionDays) ||
          Number.isNaN(auditLogsRetentionDays)
        ) {
          console.error(`Invalid quota metadata - parsed to NaN for product: ${product.id}`);
          return new Response("OK", { status: 200 });
        }

        // Update quotas based on product metadata
        await db
          .insert(schema.quotas)
          .values({
            workspaceId: ws.id,
            requestsPerMonth,
            logsRetentionDays,
            auditLogsRetentionDays,
            team: true,
          })
          .onDuplicateKeyUpdate({
            set: {
              requestsPerMonth,
              logsRetentionDays,
              auditLogsRetentionDays,
              team: true,
            },
          });

        await insertAuditLogs(db, {
          workspaceId: ws.id,
          actor: {
            type: "system",
            id: "stripe",
          },
          event: "workspace.update",
          description: `Subscription updated to ${product.name} plan.`,
          resources: [],
          context: {
            location: "",
            userAgent: undefined,
          },
        });

        // Send notification for subscription change
        if (customer && !customer.deleted && customer.email) {
          const formattedPrice = new Intl.NumberFormat("en-US", {
            style: "currency",
            currency: "USD",
          }).format(price.unit_amount / 100);

          // Send notification for any subscription update
          await alertSlackSubscriptionUpdate(
            product.name,
            formattedPrice,
            customer.email,
            customer.name || "Unknown",
          );
        }
      } catch (error) {
        console.error("Webhook error:", error);
        return new Response("Error", { status: 500 });
      }
      break;
    }
    case "customer.subscription.deleted": {
      const sub = event.data.object as Stripe.Subscription;

      const ws = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.stripeSubscriptionId, sub.id), isNull(table.deletedAtM)),
      });
      if (!ws) {
        console.error("Workspace not found for subscription:", sub.id);
        return new Response("OK", { status: 200 });
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

      // Send notification for subscription cancellation
      if (sub.customer) {
        const customer = await stripe.customers.retrieve(
          typeof sub.customer === "string" ? sub.customer : sub.customer.id,
        );

        if (customer && !customer.deleted && customer.email) {
          await alertSlackCancellation(customer.email, customer.name || "Unknown");
        }
      }
      break;
    }
    case "customer.subscription.created": {
      try {
        const sub = event.data.object as Stripe.Subscription;

        if (!sub.items?.data?.[0]?.price?.id || !sub.customer) {
          return new Response("OK");
        }

        const [price, customer] = await Promise.all([
          stripe.prices.retrieve(sub.items.data[0].price.id),
          stripe.customers.retrieve(
            typeof sub.customer === "string" ? sub.customer : sub.customer.id,
          ),
        ]);

        if (!price.product || price.unit_amount === null || price.unit_amount === undefined) {
          throw new Error("Invalid price data");
        }

        const product = await stripe.products.retrieve(
          typeof price.product === "string" ? price.product : price.product.id,
        );

        if (customer.deleted || !customer.email) {
          throw new Error("Invalid customer data");
        }

        const formattedPrice = new Intl.NumberFormat("en-US", {
          style: "currency",
          currency: "USD",
        }).format(price.unit_amount / 100);

        await alertSlack(product.name, formattedPrice, customer.email, customer.name || "Unknown");
        break;
      } catch (error) {
        console.error("Webhook error:", error);
        return new Response("Error", { status: 500 });
      }
    }

    default:
      console.warn("Incoming stripe event, that should not be received", event.type);
      break;
  }
  return new Response("OK");
};

async function alertSlack(
  product: string,
  price: string,
  email: string,
  name?: string,
): Promise<void> {
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
      blocks: [
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `:bugeyes: New customer ${name} signed up`,
          },
        },
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `A new subscription for the ${product} tier has started at a price of ${price} by ${email} :moneybag: `,
          },
        },
      ],
    }),
  }).catch((err: Error) => {
    console.error(err);
  });
}

async function alertSlackSubscriptionUpdate(
  product: string,
  price: string,
  email: string,
  name?: string,
): Promise<void> {
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
      blocks: [
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `:stonks: ${name} updated their subscription`,
          },
        },
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `Subscription updated to the ${product} tier at ${price} by ${email}q`,
          },
        },
      ],
    }),
  }).catch((err: Error) => {
    console.error(err);
  });
}

async function alertIsCancellingSubscription(
  product: string,
  price: string,
  email: string,
  name?: string,
): Promise<void> {
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
      blocks: [
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `:warning: ${name} is cancelling their subscription.`,
          },
        },
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `Subscription cancellation requested by ${email} - for ${product} at ${price}, they will be moved back to the free tier, at the end of the month. We should reach out to find out why they are cancelling.`,
          },
        },
      ],
    }),
  }).catch((err: Error) => {
    console.error(err);
  });
}

async function alertSlackCancellation(email: string, name?: string): Promise<void> {
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
      blocks: [
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `:caleb-sad: ${name} cancelled their subscription`,
          },
        },
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `Subscription cancelled by ${email} - they've been moved back to the free tier`,
          },
        },
      ],
    }),
  }).catch((err: Error) => {
    console.error(err);
  });
}
