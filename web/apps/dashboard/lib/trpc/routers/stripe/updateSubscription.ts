import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { syncSubscriptionFromStripe } from "@/lib/stripe/sync";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const updateSubscription = workspaceProcedure
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

    const updatedSub = await stripe.subscriptions.retrieve(sub.id);

    if (updatedSub.status === "incomplete" || updatedSub.status === "past_due") {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message:
          "Payment failed. Please update your payment method and try again, or stay on your current plan.",
      });
    }

    await syncSubscriptionFromStripe(stripe, ctx.workspace.id);

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

    await invalidateWorkspaceCache(ctx.tenant.id);
  });
