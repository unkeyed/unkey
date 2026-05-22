import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { getStripeClient } from "@/lib/stripe";
import { validateAndParseQuotas } from "@/lib/stripe/productUtils";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";
import { clearWorkspaceCache } from "../workspace/getCurrent";

export const createSubscription = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(
    z.object({
      productId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const stripe = getStripeClient();
    const e = stripeEnv();
    if (!e) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Stripe is not set up",
      });
    }

    // Reject any product that is not on the configured allow-list, so a
    // workspace admin can only subscribe to plans the operator has explicitly
    // exposed. Without this, a known/internal Stripe product id with $0
    // default price (or permissive quota metadata) lets an admin self-grant
    // a higher tier.
    const allowedProductIds = new Set<string>([
      ...e.STRIPE_PRODUCT_IDS_PRO,
      ...e.STRIPE_PRODUCT_IDS_ENTERPRISE,
    ]);
    if (!allowedProductIds.has(input.productId)) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Could not find product ${input.productId}.`,
      });
    }

    const product = await stripe.products.retrieve(input.productId);

    if (!product) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Could not find product ${input.productId}.`,
      });
    }

    if (!product.default_price) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Could not find product default price ${input.productId}.`,
      });
    }

    const defaultPriceId = product.default_price.toString();

    const quotas = validateAndParseQuotas(product);
    if (
      !quotas.valid ||
      quotas.requestsPerMonth === undefined ||
      quotas.logsRetentionDays === undefined ||
      quotas.auditLogsRetentionDays === undefined
    ) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: `Product ${input.productId} is missing required quota metadata.`,
      });
    }

    if (!ctx.workspace.stripeCustomerId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspaces does not have a stripe account.",
      });
    }

    if (ctx.workspace.stripeSubscriptionId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: `Customer ${ctx.workspace.stripeCustomerId} already has a subscription.`,
      });
    }

    let createdSub: Stripe.Subscription | undefined;
    try {
      await db.transaction(async (tx) => {
        const [locked] = await tx
          .select({
            stripeCustomerId: schema.workspaces.stripeCustomerId,
            stripeSubscriptionId: schema.workspaces.stripeSubscriptionId,
          })
          .from(schema.workspaces)
          .where(eq(schema.workspaces.id, ctx.workspace.id))
          .for("update");

        if (!locked) {
          throw new TRPCError({
            code: "NOT_FOUND",
            message: "Workspace could not be found.",
          });
        }
        if (!locked.stripeCustomerId) {
          throw new TRPCError({
            code: "PRECONDITION_FAILED",
            message: "Workspaces does not have a stripe account.",
          });
        }
        if (locked.stripeSubscriptionId) {
          throw new TRPCError({
            code: "PRECONDITION_FAILED",
            message: `Customer ${locked.stripeCustomerId} already has a subscription.`,
          });
        }

        const customer = await stripe.customers.retrieve(locked.stripeCustomerId);

        if (customer.deleted) {
          throw new TRPCError({
            code: "NOT_FOUND",
            message: `Customer ${locked.stripeCustomerId} could not be found.`,
          });
        }

        try {
          createdSub = await stripe.subscriptions.create({
            customer: customer.id,
            items: [
              {
                price: defaultPriceId,
              },
            ],
            billing_cycle_anchor_config: {
              day_of_month: 1,
            },
            proration_behavior: "always_invoice",
            payment_behavior: "error_if_incomplete",
          });
        } catch (err) {
          if (err instanceof Stripe.errors.StripeCardError) {
            throw new TRPCError({
              code: "BAD_REQUEST",
              message:
                err.message ||
                "Your card was declined. Please update your payment method and try again.",
            });
          }
          if (err instanceof Stripe.errors.StripeError) {
            throw new TRPCError({
              code: "BAD_REQUEST",
              message:
                err.message ||
                "Payment could not be completed. Please update your payment method and try again.",
            });
          }
          throw err;
        }

        if (createdSub.status !== "active") {
          throw new TRPCError({
            code: "BAD_REQUEST",
            message: `Subscription was created but is not active (ID: ${createdSub.id}). Please contact support.`,
          });
        }

        await tx
          .update(schema.workspaces)
          .set({
            stripeSubscriptionId: createdSub.id,
            tier: product.name,
          })
          .where(eq(schema.workspaces.id, ctx.workspace.id));

        await tx
          .insert(schema.quotas)
          .values({
            workspaceId: ctx.workspace.id,
            requestsPerMonth: quotas.requestsPerMonth,
            logsRetentionDays: quotas.logsRetentionDays,
            auditLogsRetentionDays: quotas.auditLogsRetentionDays,
            team: true,
          })
          .onDuplicateKeyUpdate({
            set: {
              requestsPerMonth: quotas.requestsPerMonth,
              logsRetentionDays: quotas.logsRetentionDays,
              auditLogsRetentionDays: quotas.auditLogsRetentionDays,
              team: true,
            },
          });

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "workspace.update",
          description: `Subscribed to ${product.name} plan`,
          resources: [],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      });
    } catch (err) {
      if (createdSub) {
        try {
          await stripe.subscriptions.cancel(createdSub.id);
        } catch (cancelErr) {
          console.error(
            `Failed to cancel orphaned subscription ${createdSub.id} after transaction failure:`,
            cancelErr,
          );
        }
      }
      throw err;
    }

    // Invalidate workspace cache after subscription creation
    await invalidateWorkspaceCache(ctx.tenant.id);

    // Also clear the tRPC workspace cache to ensure fresh data on next request
    clearWorkspaceCache(ctx.tenant.id);
  });
