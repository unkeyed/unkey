import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { getStripeClient } from "@/lib/stripe";
import { syncSubscriptionFromStripe } from "@/lib/stripe/sync";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import { clearWorkspaceCache } from "../workspace/getCurrent";

export const createSubscription = workspaceProcedure
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
          price: product.default_price.toString(),
        },
      ],
      billing_cycle_anchor_config: {
        day_of_month: 1,
      },

      proration_behavior: "always_invoice",
    });

    if (sub.status === "incomplete" || sub.status === "past_due") {
      await stripe.subscriptions.cancel(sub.id);
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message:
          "Payment failed. Please update your payment method and try again, or stay on the free plan.",
      });
    }

    const result = await fault(async () => {
      await db
        .update(schema.workspaces)
        .set({
          stripeSubscriptionId: sub.id,
        })
        .where(eq(schema.workspaces.id, ctx.workspace.id));

      await syncSubscriptionFromStripe(stripe, ctx.workspace.id);
    });

    if (result instanceof Error) {
      // Attempt to rollback subscription ID on failure
      await db
        .update(schema.workspaces)
        .set({ stripeSubscriptionId: null })
        .where(eq(schema.workspaces.id, ctx.workspace.id));
      throw result;
    }

    await insertAuditLogs(db, {
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

    await invalidateWorkspaceCache(ctx.tenant.id);
    clearWorkspaceCache(ctx.tenant.id);
  });
