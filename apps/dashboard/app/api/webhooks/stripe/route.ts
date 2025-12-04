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

interface PreviousAttributes {
  // Billing period dates (change during automated renewals)
  current_period_end?: number;
  current_period_start?: number;

  // Subscription items and pricing (change during manual updates)
  items?: {
    data?: Array<{
      current_period_end?: number;
      current_period_start?: number;
      price?: {
        id?: string;
      };
    }>;
  };

  // Other subscription properties that can change manually
  plan?: Stripe.Plan | null;
  quantity?: number;
  discount?: Stripe.Discount | null;
  cancel_at_period_end?: boolean;
  collection_method?: string;
  latest_invoice?: string | Stripe.Invoice | null;
}

function isAutomatedBillingRenewal(
  sub: Stripe.Subscription,
  previousAttributes: PreviousAttributes | undefined,
): boolean {
  // Treat as automated renewal when:
  // 1. subscription status is active
  // 2. previousAttributes exists
  // 3. Only contains billing period changes (current_period_start, current_period_end) and optionally items
  // 4. items object only contains period date changes, not price/plan changes
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

  // Check if items changed and if so, ensure it's only period date changes
  if (changedKeys.includes("items")) {
    const itemsChange = previousAttributes.items;
    if (!itemsChange || !itemsChange.data || !itemsChange.data[0]) {
      return false;
    }

    const itemChange = itemsChange.data[0];
    const itemChangedKeys = Object.keys(itemChange);

    // Define keys that indicate manual subscription changes (not automated renewals)
    const manualItemChangeKeys = ["price", "plan", "quantity", "discount"];

    // If any manual item change keys are present, this is not an automated renewal
    if (manualItemChangeKeys.some((key) => itemChangedKeys.includes(key))) {
      return false;
    }
  }

  // Define expected keys for automated renewal (period dates + optional items)
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

        // Skip database updates and notifications for automated billing renewals
        if (isAutomatedBillingRenewal(sub, previousAttributes)) {
          return new Response("Skip", { status: 201 });
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
            console.error("Error retrieving previous subscription details:", error);
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
          console.error("Workspace not found for customer:", customerId);
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
