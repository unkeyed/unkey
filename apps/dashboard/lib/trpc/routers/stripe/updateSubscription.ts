import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";
export const updateSubscription = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      oldProductId: z.string(),
      newProductId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const stripe = getStripeClient();

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

    const item = sub.items.data.find((i) => {
      const product = i.plan.product;
      return product && product.toString() === oldProduct.id;
    });

    if (!item) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: `You're not currently subscribed to ${oldProduct.id}.`,
      });
    }

    if (!item.id) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Subscription item is missing an ID.",
      });
    }

    if (!newProduct.default_price) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: `Product ${newProduct.id} is missing a default price.`,
      });
    }

    await stripe.subscriptionItems.update(item.id, {
      price: newProduct.default_price.toString(),
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

    await insertAuditLogs(db, {
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

    // Invalidate workspace cache after subscription update
    await invalidateWorkspaceCache(ctx.tenant.id);
  });
