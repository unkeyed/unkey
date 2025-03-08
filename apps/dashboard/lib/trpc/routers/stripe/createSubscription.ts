import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { auth, t } from "../../trpc";
export const createSubscription = t.procedure
  .use(auth)
  .input(
    z.object({
      productId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const e = stripeEnv();
    if (!e) {
      throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: "Stripe is not set up" });
    }

    const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
      apiVersion: "2023-10-16",
      typescript: true,
    });

    const product = await stripe.products.retrieve(input.productId);

    if (!product) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Could not find product ${input.productId}.`,
      });
    }

    if (!ctx.workspace.stripeCustomerId) {
      const baseUrl = process.env.VERCEL_URL
        ? process.env.VERCEL_TARGET_ENV === "production"
          ? "https://app.unkey.com"
          : `https://${process.env.VERCEL_URL}`
        : "http://localhost:3000";

      const session = await stripe.checkout.sessions.create({
        client_reference_id: ctx.workspace.id,
        billing_address_collection: "auto",
        mode: "setup",
        success_url: `${baseUrl}/settings/billing/stripe/checkout-success?session_id={CHECKOUT_SESSION_ID}&product_id=${input.productId}`,
        currency: "USD",
        customer_creation: "always",
      });

      if (!session.url) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "We're unable to generate a checkout session.",
        });
      }

      return {
        url: session.url,
      };
    }
    if (ctx.workspace.stripeSubscriptionId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: `Customer ${ctx.workspace.stripeCustomerId} already has a subscription.`,
      });
    }

    const customer = await stripe.customers.retrieve(ctx.workspace.stripeCustomerId);
    if (!customer) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Customer ${ctx.workspace.stripeCustomerId} could not be found.`,
      });
    }

    const sub = await stripe.subscriptions.create({
      customer: customer.id,
      items: [
        {
          price: product.default_price!.toString(),
        },
      ],
      billing_cycle_anchor_config: {
        day_of_month: 1,
      },

      proration_behavior: "always_invoice",
    });
    await db
      .update(schema.workspaces)
      .set({
        stripeSubscriptionId: sub.id,
      })
      .where(eq(schema.workspaces.id, ctx.workspace.id));
    await db
      .insert(schema.quotas)
      .values({
        workspaceId: ctx.workspace.id,
        requestsPerMonth: Number.parseInt(product.metadata.quota_requests_per_month),
        logsRetentionDays: Number.parseInt(product.metadata.quota_logs_retention_days),
        auditLogsRetentionDays: Number.parseInt(product.metadata.quota_audit_logs_retention_days),
        team: true,
      })
      .onDuplicateKeyUpdate({
        set: {
          requestsPerMonth: Number.parseInt(product.metadata.quota_requests_per_month),
          logsRetentionDays: Number.parseInt(product.metadata.quota_logs_retention_days),
          auditLogsRetentionDays: Number.parseInt(product.metadata.quota_audit_logs_retention_days),
          team: true,
        },
      });

    await insertAuditLogs(db, ctx.workspace.auditLogBucket.id, {
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
