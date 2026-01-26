import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { formatPrice } from "@/lib/fmt";
import { isPaymentRecovery, isPaymentRecoveryUpdate } from "@/lib/stripe/paymentUtils";
import {
  isAutomatedBillingRenewal,
  isCardUpdateOnly,
  isPaymentFailureRelatedUpdate,
} from "@/lib/stripe/subscriptionUtils";
import {
  alertIsCancellingSubscription,
  alertPaymentFailed,
  alertPaymentRecovered,
  alertSubscriptionCancelled,
  alertSubscriptionCreation,
  alertSubscriptionUpdate,
} from "@/lib/utils/slackAlerts";
import Stripe from "stripe";

export const runtime = "nodejs";

export const POST = async (req: Request): Promise<Response> => {
  const signature = req.headers.get("stripe-signature");
  if (!signature) {
    console.error("Webhook signature validation failed: Missing stripe-signature header");
    return new Response("Webhook signature missing", { status: 400 });
  }

  const e = stripeEnv();

  if (!e) {
    console.error(
      "Stripe environment configuration is missing. Check that STRIPE_SECRET_KEY and other required Stripe environment variables are properly set.",
    );
    return new Response("Server configuration error", { status: 500 });
  }

  const stripeSecretKey = stripeEnv()?.STRIPE_SECRET_KEY;
  if (!stripeSecretKey) {
    console.error(
      "STRIPE_SECRET_KEY environment variable is not set. This is required for Stripe API operations.",
    );
    return new Response("Server configuration error", { status: 500 });
  }

  const stripe = new Stripe(stripeSecretKey, {
    apiVersion: "2023-10-16",
    typescript: true,
  });

  let event: Stripe.Event;
  let requestBody: string;

  try {
    requestBody = await req.text();
  } catch (error) {
    console.error("Failed to read request body:", error);
    return new Response("Error", { status: 400 });
  }

  try {
    event = stripe.webhooks.constructEvent(requestBody, signature, e.STRIPE_WEBHOOK_SECRET);
  } catch (error) {
    console.error("Webhook signature validation failed:", error);
    return new Response("Error", { status: 400 });
  }
  switch (event.type) {
    case "customer.subscription.updated": {
      try {
        const sub = event.data.object as Stripe.Subscription;

        const ws = await db.query.workspaces.findFirst({
          where: (table, { and, eq, isNull }) =>
            and(eq(table.stripeSubscriptionId, sub.id), isNull(table.deletedAtM)),
        });
        if (!ws) {
          console.error("Workspace not found for subscription:", {
            subscriptionId: sub.id,
            eventId: event.id,
          });
          return new Response("OK", { status: 200 });
        }

        const previousAttributes = event.data.previous_attributes;

        if (isAutomatedBillingRenewal(sub, previousAttributes)) {
          return new Response("OK", { status: 200 });
        }

        if (isPaymentFailureRelatedUpdate(sub, previousAttributes)) {
          return new Response("OK", { status: 200 });
        }

        const isRecovery = await isPaymentRecoveryUpdate(stripe, sub, previousAttributes, event);
        if (isRecovery) {
          return new Response("OK", { status: 200 });
        }

        if (isCardUpdateOnly(sub, previousAttributes)) {
          return new Response("OK", { status: 200 });
        }

        // Map Stripe status to allowed database values
        const allowedStatuses = [
          "active",
          "past_due",
          "canceled",
          "unpaid",
          "trialing",
          "incomplete",
          "incomplete_expired",
        ] as const;
        type AllowedStatus = (typeof allowedStatuses)[number];
        const isAllowedStatus = (value: unknown): value is AllowedStatus =>
          allowedStatuses.includes(value as AllowedStatus);
        const subscriptionStatus = isAllowedStatus(sub.status) ? sub.status : "canceled";

        await db
          .update(schema.workspaces)
          .set({
            subscriptionStatus,
          })
          .where(eq(schema.workspaces.id, ws.id));

        if (sub.cancel_at) {
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

          if (customer && !customer.deleted && customer.email) {
            const formattedPrice = formatPrice(price.unit_amount);
            await alertIsCancellingSubscription(
              product.name,
              formattedPrice,
              customer.email,
              customer.name || "Unknown",
            );
          }
          return new Response("OK");
        }

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
            console.error("Error retrieving previous subscription details:", {
              error,
              eventId: event.id,
              subscriptionId: sub.id,
            });
          }
        }

        if (customer && !customer.deleted && customer.email) {
          const formattedPrice = formatPrice(price.unit_amount);

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
        console.error("Subscription update webhook error:", {
          error:
            error instanceof Error
              ? {
                  message: error.message,
                  stack: error.stack,
                  name: error.name,
                }
              : error,
          eventId: event.id,
          eventType: event.type,
        });
        return new Response("Error", { status: 500 });
      }
      break;
    }
    case "customer.subscription.deleted": {
      try {
        const sub = event.data.object as Stripe.Subscription;

        const ws = await db.query.workspaces.findFirst({
          where: (table, { and, eq, isNull }) =>
            and(eq(table.stripeSubscriptionId, sub.id), isNull(table.deletedAtM)),
        });
        if (!ws) {
          console.error("Workspace not found for subscription:", {
            subscriptionId: sub.id,
            eventId: event.id,
          });
          return new Response("OK", { status: 200 });
        }

        await db
          .update(schema.workspaces)
          .set({
            subscriptionStatus: "canceled",
          })
          .where(eq(schema.workspaces.id, ws.id));

        if (sub.customer) {
          try {
            const customer = await stripe.customers.retrieve(
              typeof sub.customer === "string" ? sub.customer : sub.customer.id,
            );

            if (customer && !customer.deleted && customer.email) {
              await alertSubscriptionCancelled(customer.email, customer.name || "Unknown");
            }
          } catch (customerError) {
            console.error("Failed to retrieve customer for subscription cancellation alert:", {
              error: customerError,
              subscriptionId: sub.id,
              eventId: event.id,
            });
          }
        }
      } catch (error) {
        console.error("Subscription deletion webhook error:", {
          error:
            error instanceof Error
              ? {
                  message: error.message,
                  stack: error.stack,
                  name: error.name,
                }
              : error,
          eventId: event.id,
          eventType: event.type,
        });
        return new Response("Error", { status: 500 });
      }
      break;
    }
    case "customer.subscription.created": {
      try {
        const sub = event.data.object as Stripe.Subscription;

        if (!sub.items?.data?.[0]?.price?.id || !sub.customer) {
          return new Response("OK");
        }

        if (sub.status === "incomplete" || sub.status === "incomplete_expired") {
          return new Response("OK", { status: 200 });
        }

        const customerId = typeof sub.customer === "string" ? sub.customer : sub.customer.id;
        const ws = await db.query.workspaces.findFirst({
          where: (table, { and, eq, isNull }) =>
            and(eq(table.stripeCustomerId, customerId), isNull(table.deletedAtM)),
        });

        if (!ws) {
          console.error("Workspace not found for customer:", {
            customerId,
            eventId: event.id,
          });
          return new Response("OK", { status: 200 });
        }

        // Map Stripe status to allowed database values
        const allowedStatuses = [
          "active",
          "past_due",
          "canceled",
          "unpaid",
          "trialing",
          "incomplete",
          "incomplete_expired",
        ] as const;
        type AllowedStatus = (typeof allowedStatuses)[number];
        const isAllowedStatus = (value: unknown): value is AllowedStatus =>
          allowedStatuses.includes(value as AllowedStatus);
        const subscriptionStatus = isAllowedStatus(sub.status) ? sub.status : "canceled";

        await db
          .update(schema.workspaces)
          .set({
            subscriptionStatus,
          })
          .where(eq(schema.workspaces.id, ws.id));

        const [price, customer] = await Promise.all([
          stripe.prices.retrieve(sub.items.data[0].price.id),
          stripe.customers.retrieve(customerId),
        ]);

        if (!price.product || price.unit_amount === null || price.unit_amount === undefined) {
          return new Response("OK");
        }

        const product = await stripe.products.retrieve(
          typeof price.product === "string" ? price.product : price.product.id,
        );

        if (customer.deleted || !customer.email) {
          return new Response("OK");
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
        console.error("Subscription creation webhook error:", {
          error:
            error instanceof Error
              ? {
                  message: error.message,
                  stack: error.stack,
                  name: error.name,
                }
              : error,
          eventId: event.id,
          eventType: event.type,
        });
        return new Response("Error", { status: 500 });
      }
    }

    case "invoice.payment_failed": {
      try {
        const invoice = event.data.object as Stripe.Invoice;

        if (!invoice || typeof invoice !== "object") {
          console.error("Payment failed event received with invalid invoice data structure");
          return new Response("Invalid event data", { status: 400 });
        }

        if (!invoice.customer) {
          console.warn("Payment failed event received without customer information", {
            invoiceId: invoice.id,
            eventId: event.id,
          });
          return new Response("OK", { status: 200 });
        }

        const customerId =
          typeof invoice.customer === "string" ? invoice.customer : invoice.customer.id;

        const ws = await db.query.workspaces.findFirst({
          where: (table, { and, eq, isNull }) =>
            and(eq(table.stripeCustomerId, customerId), isNull(table.deletedAtM)),
        });

        if (ws) {
          await db
            .update(schema.workspaces)
            .set({
              paymentFailedAt: Date.now(),
              subscriptionStatus: "past_due",
            })
            .where(eq(schema.workspaces.id, ws.id));
        }

        let customer: Stripe.Customer | Stripe.DeletedCustomer;

        try {
          customer = await stripe.customers.retrieve(customerId);
        } catch (customerError) {
          console.error("Failed to retrieve customer for payment failure event:", {
            error: customerError,
            customerId,
            invoiceId: invoice.id,
            eventId: event.id,
          });
          return new Response("OK", { status: 200 });
        }

        if (customer.deleted || !("email" in customer) || !customer.email) {
          return new Response("OK", { status: 200 });
        }

        const amount = invoice.amount_due || 0;
        const currency = invoice.currency || "usd";

        if (amount < 0) {
          console.warn("Payment failed event with negative amount", {
            amount,
            invoiceId: invoice.id,
            eventId: event.id,
          });
        }

        try {
          const customerEmail = (customer as Stripe.Customer).email;
          if (customerEmail) {
            await alertPaymentFailed(
              customerEmail,
              (customer as Stripe.Customer).name || "Unknown",
              amount,
              currency,
            );
          }
        } catch (alertError) {
          console.error("Failed to send payment failure alert:", {
            error: alertError,
            customerEmail: (customer as Stripe.Customer).email,
            invoiceId: invoice.id,
            eventId: event.id,
          });
          return new Response("Alert failed but event processed", { status: 200 });
        }

        return new Response("OK", { status: 200 });
      } catch (error) {
        console.error("Error processing payment failure webhook:", {
          error:
            error instanceof Error
              ? {
                  message: error.message,
                  stack: error.stack,
                  name: error.name,
                }
              : error,
          eventId: event.id,
          eventType: event.type,
        });

        return new Response("Error processing payment failure", { status: 200 });
      }
    }

    case "invoice.payment_succeeded": {
      try {
        const invoice = event.data.object as Stripe.Invoice;

        if (!invoice || typeof invoice !== "object") {
          console.error("Payment success event received with invalid invoice data structure");
          return new Response("Invalid event data", { status: 400 });
        }

        if (!invoice.customer) {
          console.warn("Payment success event received without customer information", {
            invoiceId: invoice.id,
            eventId: event.id,
          });
          return new Response("OK", { status: 200 });
        }

        const customerId =
          typeof invoice.customer === "string" ? invoice.customer : invoice.customer.id;

        const ws = await db.query.workspaces.findFirst({
          where: (table, { and, eq, isNull }) =>
            and(eq(table.stripeCustomerId, customerId), isNull(table.deletedAtM)),
        });

        if (ws && invoice.subscription) {
          const subscriptionId =
            typeof invoice.subscription === "string"
              ? invoice.subscription
              : invoice.subscription.id;

          let subscription: Stripe.Subscription;
          try {
            subscription = await stripe.subscriptions.retrieve(subscriptionId);
          } catch (subError) {
            console.error("Failed to retrieve subscription for payment success event:", {
              error: subError,
              subscriptionId,
              invoiceId: invoice.id,
              eventId: event.id,
            });
            return new Response("OK", { status: 200 });
          }

          if (subscription.status === "active") {
            await db
              .update(schema.workspaces)
              .set({
                paymentFailedAt: null,
                paymentFailureNotifiedAt: null,
                subscriptionStatus: "active",
              })
              .where(eq(schema.workspaces.id, ws.id));
          }
        }

        let customer: Stripe.Customer | Stripe.DeletedCustomer;

        try {
          customer = await stripe.customers.retrieve(customerId);
        } catch (customerError) {
          console.error("Failed to retrieve customer for payment success event:", {
            error: customerError,
            customerId,
            invoiceId: invoice.id,
            eventId: event.id,
          });
          return new Response("OK", { status: 200 });
        }

        if (customer.deleted || !("email" in customer) || !customer.email) {
          return new Response("OK", { status: 200 });
        }

        let isRecovery = false;

        try {
          isRecovery = await isPaymentRecovery(stripe, event);
        } catch (recoveryError) {
          console.error("Failed to determine payment recovery status:", {
            error: recoveryError,
            invoiceId: invoice.id,
            eventId: event.id,
            customerEmail: customer.email,
          });
          isRecovery = false;
        }

        if (isRecovery) {
          const amount = invoice.amount_paid || 0;
          const currency = invoice.currency || "usd";

          if (amount < 0) {
            console.warn("Payment success event with negative amount", {
              amount,
              invoiceId: invoice.id,
              eventId: event.id,
            });
          }

          const customerEmail = (customer as Stripe.Customer).email;
          if (customerEmail) {
            try {
              await alertPaymentRecovered(
                customerEmail,
                (customer as Stripe.Customer).name || "Unknown",
                amount,
                currency,
              );
            } catch (alertError) {
              console.error("Failed to send payment recovery alert:", {
                error: alertError,
                customerEmail,
                invoiceId: invoice.id,
                eventId: event.id,
              });
              return new Response("Alert failed but event processed", { status: 200 });
            }
          }
        }

        return new Response("OK", { status: 200 });
      } catch (error) {
        console.error("Error processing payment success webhook:", {
          error:
            error instanceof Error
              ? {
                  message: error.message,
                  stack: error.stack,
                  name: error.name,
                }
              : error,
          eventId: event.id,
          eventType: event.type,
        });

        return new Response("Error processing payment success", { status: 200 });
      }
    }

    default:
      break;
  }
  return new Response("OK");
};
