import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { auth, t } from "../../trpc";
export const updateSubscription = t.procedure
  .use(auth)
  .input(
    z.object({
      oldProductId: z.string(),
      newProductId: z.string(),
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

    if (!ctx.workspace.stripeCustomerId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspace doesn't have a stripe customer id.",
      });
    }
    if (!ctx.workspace.stripeSubscriptionId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspace doesn't have a stripe subscrption id.",
      });
    }

    const [oldProduct, newProduct] = await Promise.all([
      stripe.products.retrieve(input.oldProductId),
      stripe.products.retrieve(input.newProductId),
    ]);

    if (!oldProduct) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Could not find product ${input.oldProductId}.`,
      });
    }

    if (!newProduct) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Could not find product ${input.newProductId}.`,
      });
    }

    const sub = await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId);

    if (!sub) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Could not find subscription ${ctx.workspace.stripeSubscriptionId}.`,
      });
    }

    const item = sub.items.data.find((i) => i.plan.product!.toString() === oldProduct.id);
    if (!item) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: `You're not currently subscribed to ${oldProduct.id}.`,
      });
    }

    await stripe.subscriptionItems.update(item.id!, {
      price: newProduct.default_price!.toString(),
      proration_behavior: "always_invoice",
    });

    if (sub.cancel_at) {
      await stripe.subscriptions.update(sub.id, {
        cancel_at_period_end: false,
      });
    }

    await db
      .update(schema.workspaces)
      .set({
        tier: newProduct.name,
      })
      .where(eq(schema.workspaces.id, ctx.workspace.id));

    await db
      .insert(schema.quotas)
      .values({
        workspaceId: ctx.workspace.id,
        requestsPerMonth: Number.parseInt(newProduct.metadata.quota_requests_per_month),
        logsRetentionDays: Number.parseInt(newProduct.metadata.quota_logs_retention_days),
        auditLogsRetentionDays: Number.parseInt(
          newProduct.metadata.quota_audit_logs_retention_days,
        ),
        team: true,
      })
      .onDuplicateKeyUpdate({
        set: {
          requestsPerMonth: Number.parseInt(newProduct.metadata.quota_requests_per_month),
          logsRetentionDays: Number.parseInt(newProduct.metadata.quota_logs_retention_days),
          auditLogsRetentionDays: Number.parseInt(
            newProduct.metadata.quota_audit_logs_retention_days,
          ),
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
      description: `Switched to ${newProduct.name} plan.`,
      resources: [],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });
  });
