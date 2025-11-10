import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { formatPrice } from "@/lib/fmt";
import { freeTierQuotas } from "@/lib/quotas";
import {
  alertIsCancellingSubscription,
  alertSubscriptionCancelled,
  alertSubscriptionCreation,
  alertSubscriptionUpdate,
} from "@/lib/utils/slackAlerts";
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

        const previousAttributes = event.data.previous_attributes;

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

        /**
         * In our case, when a user cancels their subscription, it's not in effect until the beginning of the next month.
         * So we get a subscription updated event, which we should handle accordingly.
         */
        if (sub.cancel_at) {
          if (customer && !customer.deleted && customer.email) {
            const formattedPrice = formatPrice(price.unit_amount!);
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

        /**
         * To make the updates more useful, we detect if they are downgrading or upgrading their subscription
         * We can then send a good or bad update based upon it.
         */
        let changeType = "updated";
        let previousTier: string | undefined;

        if (previousAttributes?.items?.data?.[0]?.price?.id) {
          try {
            const previousPrice = await stripe.prices.retrieve(
              previousAttributes.items.data[0].price.id,
            );

            if (previousPrice.product && previousPrice.unit_amount !== null) {
              const previousProduct = await stripe.products.retrieve(
                typeof previousPrice.product === "string"
                  ? previousPrice.product
                  : previousPrice.product.id,
              );

              previousTier = previousProduct.name;

              // Compare amounts to determine upgrade/downgrade
              const currentAmount = price.unit_amount;
              const previousAmount = previousPrice.unit_amount;

              if (currentAmount !== previousAmount && previousAmount !== null) {
                if (currentAmount > previousAmount) {
                  changeType = "upgraded";
                } else if (currentAmount < previousAmount) {
                  changeType = "downgraded";
                }
              }
            }
          } catch (error) {
            console.error("Error retrieving previous subscription details:", error);
          }
        }

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

        if (customer && !customer.deleted && customer.email) {
          const formattedPrice = formatPrice(price.unit_amount);

          // Send notification for any subscription update
          await alertSubscriptionUpdate(
            product.name,
            formattedPrice,
            customer.email,
            customer.name || "Unknown",
            changeType,
            previousTier,
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
          await alertSubscriptionCancelled(customer.email, customer.name || "Unknown");
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

        const formattedPrice = formatPrice(price.unit_amount);

        await alertSubscriptionCreation(
          product.name,
          formattedPrice,
          customer.email,
          customer.name || "Unknown",
        );
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
