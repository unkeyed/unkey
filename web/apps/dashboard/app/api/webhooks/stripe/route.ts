import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { formatPrice } from "@/lib/fmt";
import { freeTierQuotas } from "@/lib/quotas";
import { isPaymentRecovery } from "@/lib/utils/paymentRecoveryDetection";
import {
  alertIsCancellingSubscription,
  alertPaymentFailed,
  alertPaymentRecovered,
  alertSubscriptionCancelled,
  alertSubscriptionCreation,
  alertSubscriptionUpdate,
} from "@/lib/utils/slackAlerts";
import Stripe from "stripe";

interface PreviousAttributes {
  // Billing period dates (change during automated renewals)
  current_period_end?: number;
  current_period_start?: number;

  // Subscription items and pricing (change during manual updates)
  items?: {
    data?: Partial<Stripe.SubscriptionItem>[];
  };

  // Other subscription properties that can change manually
  plan?: Stripe.Plan | null;
  quantity?: number;
  discount?: Stripe.Discount | null;
  cancel_at_period_end?: boolean;
  collection_method?: string;
  latest_invoice?: string | Stripe.Invoice | null;

  // Status changes (can indicate payment failures)
  status?: Stripe.Subscription.Status;
}

function isPaymentFailureRelatedUpdate(
  sub: Stripe.Subscription,
  previousAttributes: PreviousAttributes | undefined,
): boolean {
  // Detect if subscription update is due to payment failure
  // This happens when:
  // 1. Subscription status changed to past_due, unpaid, or incomplete
  // 2. Latest invoice changed (indicating a payment attempt)
  // 3. No manual changes to pricing, plan, or other subscription settings

  if (!previousAttributes) {
    return false;
  }

  const changedKeys = Object.keys(previousAttributes);

  // Check if status changed to a payment-failure-related status
  const paymentFailureStatuses = ["past_due", "unpaid", "incomplete"];
  const statusChanged =
    changedKeys.includes("status") && paymentFailureStatuses.includes(sub.status);

  // Check if latest_invoice changed (indicates payment processing)
  const invoiceChanged = changedKeys.includes("latest_invoice");

  // Define keys that indicate manual changes (not payment-related)
  const manualChangeKeys = [
    "cancel_at_period_end",
    "collection_method",
    "plan",
    "quantity",
    "discount",
    "items", // pricing/plan changes
  ];

  // If any manual change keys are present, this is not a payment failure update
  const hasManualChanges = manualChangeKeys.some((key) => changedKeys.includes(key));

  // Consider it a payment failure update if:
  // - Status changed to payment failure status, OR
  // - Latest invoice changed without manual subscription changes
  return (statusChanged || invoiceChanged) && !hasManualChanges;
}

function isAutomatedBillingRenewal(
  sub: Stripe.Subscription,
  previousAttributes: PreviousAttributes | undefined,
): boolean {
  // Treat as automated renewal when:
  // 1. subscription status is active
  // 2. previousAttributes exists
  // 3. Only contains billing period changes (current_period_start, current_period_end) and optionally items/latest_invoice
  // 4. If items changed, only the period dates within items actually changed (not price/plan/quantity)
  // 5. cancel_at_period_end and collection_method are not present among keys

  if (sub.status !== "active" || !previousAttributes) {
    return false;
  }

  // Get all keys that changed in previousAttributes
  const changedKeys = Object.keys(previousAttributes);

  // Define keys that indicate manual changes (not automated renewals)
  const manualChangeKeys = [
    "cancel_at_period_end",
    "collection_method",
    "plan",
    "quantity",
    "discount",
  ];

  // If any manual change keys are present, this is not an automated renewal
  if (manualChangeKeys.some((key) => changedKeys.includes(key))) {
    return false;
  }

  // Check if items changed and verify only period dates changed
  if (changedKeys.includes("items")) {
    const itemsChange = previousAttributes.items;
    if (!itemsChange || !itemsChange.data || !itemsChange.data[0] || !sub.items?.data?.[0]) {
      return false;
    }

    const previousItem = itemsChange.data[0];
    const currentItem = sub.items.data[0];

    // Check if price, plan, or quantity actually changed by comparing current vs previous
    if (
      previousItem.price?.id !== currentItem.price?.id ||
      previousItem.plan?.id !== currentItem.plan?.id ||
      previousItem.quantity !== currentItem.quantity
    ) {
      return false;
    }
  }

  // Define expected keys for automated renewal (period dates + optional items/latest_invoice)
  const allowedKeys = ["current_period_start", "current_period_end", "items", "latest_invoice"];

  // Check if all changed keys are allowed for automated renewals
  const hasOnlyAllowedKeys = changedKeys.every((key) => allowedKeys.includes(key));

  return hasOnlyAllowedKeys;
}

function validateAndParseQuotas(product: Stripe.Product): {
  valid: boolean;
  requestsPerMonth?: number;
  logsRetentionDays?: number;
  auditLogsRetentionDays?: number;
} {
  const requiredMetadata = [
    "quota_requests_per_month",
    "quota_logs_retention_days",
    "quota_audit_logs_retention_days",
  ];

  for (const field of requiredMetadata) {
    if (!product.metadata[field]) {
      console.error(`Missing required metadata field: ${field} for product: ${product.id}`);
      return { valid: false };
    }
  }

  const requestsPerMonth = Number.parseInt(product.metadata.quota_requests_per_month);
  const logsRetentionDays = Number.parseInt(product.metadata.quota_logs_retention_days);
  const auditLogsRetentionDays = Number.parseInt(product.metadata.quota_audit_logs_retention_days);

  if (
    Number.isNaN(requestsPerMonth) ||
    Number.isNaN(logsRetentionDays) ||
    Number.isNaN(auditLogsRetentionDays)
  ) {
    console.error(`Invalid quota metadata - parsed to NaN for product: ${product.id}`);
    return { valid: false };
  }

  return {
    valid: true,
    requestsPerMonth,
    logsRetentionDays,
    auditLogsRetentionDays,
  };
}

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
    return new Response("Error ", { status: 400 });
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

        // Skip database updates and notifications for automated billing renewals
        if (isAutomatedBillingRenewal(sub, previousAttributes)) {
          return new Response("OK", { status: 201 });
        }

        // Skip database updates and notifications for payment failure related updates
        // Payment failures are handled by the invoice.payment_failed webhook
        if (isPaymentFailureRelatedUpdate(sub, previousAttributes)) {
          console.info(
            "Skipping subscription update due to payment failure - handled by payment webhook",
            {
              subscriptionId: sub.id,
              eventId: event.id,
              subscriptionStatus: sub.status,
              previousAttributes: Object.keys(previousAttributes || {}),
            },
          );
          return new Response("OK", { status: 201 });
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

        /**
         * In our case, when a user cancels their subscription, it's not in effect until the beginning of the next month.
         * So we get a subscription updated event, which we should handle accordingly.
         */
        if (sub.cancel_at) {
          if (customer && !customer.deleted && customer.email) {
            const formattedPrice = formatPrice(price.unit_amount);
            await alertIsCancellingSubscription(
              product.name,
              formattedPrice,
              customer.email,
              customer.name || "Unknown",
            );

            return new Response("OK");
          }
        }

        // Validate and parse quotas
        const quotas = validateAndParseQuotas(product);
        if (!quotas.valid) {
          return new Response("OK", { status: 200 });
        }

        const { requestsPerMonth, logsRetentionDays, auditLogsRetentionDays } = quotas;

        // Update quotas and workspace tier
        await db.transaction(async (tx) => {
          await tx
            .update(schema.workspaces)
            .set({
              tier: product.name,
            })
            .where(eq(schema.workspaces.id, ws.id));

          await tx
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

          await insertAuditLogs(tx, {
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
        });

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
            console.error("Error retrieving previous subscription details:", {
              error,
              eventId: event.id,
              subscriptionId: sub.id,
            });
          }
        }

        // Send notification for subscription update
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
            stripeSubscriptionId: null,
            tier: "Free",
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
            // Continue without sending alert rather than failing
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

        if (customer.deleted || !customer.email) {
          return new Response("OK");
        }

        // Find workspace by stripe customer ID
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

        // Validate and parse quotas
        const quotas = validateAndParseQuotas(product);
        if (!quotas.valid) {
          return new Response("OK", { status: 200 });
        }

        const { requestsPerMonth, logsRetentionDays, auditLogsRetentionDays } = quotas;

        // Update workspace, quotas, and audit log in a transaction
        await db.transaction(async (tx) => {
          await tx
            .update(schema.workspaces)
            .set({
              stripeSubscriptionId: sub.id,
              tier: product.name,
            })
            .where(eq(schema.workspaces.id, ws.id));

          await tx
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

          await insertAuditLogs(tx, {
            workspaceId: ws.id,
            actor: {
              type: "system",
              id: "stripe",
            },
            event: "workspace.update",
            description: `Subscription created for ${product.name} plan.`,
            resources: [],
            context: {
              location: "",
              userAgent: undefined,
            },
          });
        });

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

        // Validate invoice data structure
        if (!invoice || typeof invoice !== "object") {
          console.error("Payment failed event received with invalid invoice data structure");
          return new Response("Invalid event data", { status: 400 });
        }

        // Extract customer information from the invoice
        if (!invoice.customer) {
          console.warn("Payment failed event received without customer information", {
            invoiceId: invoice.id,
            eventId: event.id,
          });
          return new Response("OK", { status: 200 });
        }

        let customer: Stripe.Customer | Stripe.DeletedCustomer;

        try {
          // Get customer details from Stripe with timeout handling
          customer = await stripe.customers.retrieve(
            typeof invoice.customer === "string" ? invoice.customer : invoice.customer.id,
          );
        } catch (customerError) {
          console.error("Failed to retrieve customer for payment failure event:", {
            error: customerError,
            customerId:
              typeof invoice.customer === "string" ? invoice.customer : invoice.customer.id,
            invoiceId: invoice.id,
            eventId: event.id,
          });
          // Continue processing without customer details rather than failing completely
          return new Response("OK", { status: 200 });
        }

        if (customer.deleted || !("email" in customer) || !customer.email) {
          console.warn("Payment failed event for deleted customer or customer without email", {
            customerId: customer.id,
            deleted: customer.deleted,
            hasEmail: "email" in customer && !!customer.email,
            invoiceId: invoice.id,
            eventId: event.id,
          });
          return new Response("OK", { status: 200 });
        }

        // Extract payment failure details with validation
        const amount = invoice.amount_due || 0;
        const currency = invoice.currency || "usd";
        const failureReason = invoice.last_finalization_error?.message;

        // Validate amount and currency
        if (amount < 0) {
          console.warn("Payment failed event with negative amount", {
            amount,
            invoiceId: invoice.id,
            eventId: event.id,
          });
        }

        try {
          // Send payment failure alert without triggering subscription updates
          const customerEmail = (customer as Stripe.Customer).email;
          if (customerEmail) {
            await alertPaymentFailed(
              customerEmail,
              (customer as Stripe.Customer).name || "Unknown",
              amount,
              currency,
              failureReason,
            );
          }
        } catch (alertError) {
          console.error("Failed to send payment failure alert:", {
            error: alertError,
            customerEmail: (customer as Stripe.Customer).email,
            invoiceId: invoice.id,
            eventId: event.id,
          });
          // Don't fail the webhook if alert fails - return success to prevent retries
          return new Response("Alert failed but event processed", { status: 200 });
        }

        // Return success immediately to prevent fall-through to other webhook handlers
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

        // Return 200 to prevent Stripe from retrying, but log the error
        // This ensures payment processing errors don't affect other webhook types
        return new Response("Error processing payment failure", { status: 200 });
      }
    }

    case "invoice.payment_succeeded": {
      try {
        const invoice = event.data.object as Stripe.Invoice;

        // Validate invoice data structure
        if (!invoice || typeof invoice !== "object") {
          console.error("Payment success event received with invalid invoice data structure");
          return new Response("Invalid event data", { status: 400 });
        }

        // Extract customer information from the invoice
        if (!invoice.customer) {
          console.warn("Payment success event received without customer information", {
            invoiceId: invoice.id,
            eventId: event.id,
          });
          return new Response("OK", { status: 200 });
        }

        let customer: Stripe.Customer | Stripe.DeletedCustomer;

        try {
          // Get customer details from Stripe with timeout handling
          customer = await stripe.customers.retrieve(
            typeof invoice.customer === "string" ? invoice.customer : invoice.customer.id,
          );
        } catch (customerError) {
          console.error("Failed to retrieve customer for payment success event:", {
            error: customerError,
            customerId:
              typeof invoice.customer === "string" ? invoice.customer : invoice.customer.id,
            invoiceId: invoice.id,
            eventId: event.id,
          });
          // Continue processing without customer details rather than failing completely
          return new Response("OK", { status: 200 });
        }

        if (customer.deleted || !("email" in customer) || !customer.email) {
          console.warn("Payment success event for deleted customer or customer without email", {
            customerId: customer.id,
            deleted: customer.deleted,
            hasEmail: "email" in customer && !!customer.email,
            invoiceId: invoice.id,
            eventId: event.id,
          });
          return new Response("OK", { status: 200 });
        }

        let isRecovery = false;

        try {
          // Use recovery detection logic to determine if success follows failure
          isRecovery = await isPaymentRecovery(stripe, event);
        } catch (recoveryError) {
          console.error("Failed to determine payment recovery status:", {
            error: recoveryError,
            invoiceId: invoice.id,
            eventId: event.id,
            customerEmail: customer.email,
          });
          // Assume not a recovery if detection fails to avoid false positives
          isRecovery = false;
        }

        // Send recovery alert only when appropriate (after previous failures)
        if (isRecovery) {
          const amount = invoice.amount_paid || 0;
          const currency = invoice.currency || "usd";

          // Validate amount and currency
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
              // Don't fail the webhook if alert fails - return success to prevent retries
              return new Response("Alert failed but event processed", { status: 200 });
            }
          }
        }

        // Return success immediately to prevent fall-through to other webhook handlers
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

        // Return 200 to prevent Stripe from retrying, but log the error
        // This ensures payment processing errors don't affect other webhook types
        return new Response("Error processing payment success", { status: 200 });
      }
    }

    default:
      console.warn("Incoming stripe event that should not be received:", {
        eventType: event.type,
        eventId: event.id,
      });
      break;
  }
  return new Response("OK");
};
