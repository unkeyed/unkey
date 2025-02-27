import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { auth, t } from "../../trpc";
export const changeWorkspacePlan = t.procedure
  .use(auth)
  .input(
    z.object({
      priceId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const env = stripeEnv();
    if (!env) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "stripe env not set",
      });
    }
    const stripe = new Stripe(env.STRIPE_SECRET_KEY, {
      apiVersion: "2023-10-16",
      typescript: true,
    });

    if (!ctx.workspace.stripeCustomerId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspace has no stripe customer id",
      });
    }

    if (!ctx.workspace.stripeSubscriptionId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspace has no stripe subscription id",
      });
    }

    const subscription = await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId);

    // remove old price
    const oldPrice = subscription.items.data.find(
      (i) => i.price.product === env.STRIPE_PRODUCT_ID_SCALE_PLAN,
    );

    const price = await stripe.prices.retrieve(input.priceId);
    if (!price) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Price does not exist",
      });
    }

    await stripe.subscriptions.update(ctx.workspace.stripeSubscriptionId, {
      items: [
        {
          id: oldPrice?.id,
          price: input.priceId,
        },
      ],
    });

    await db.transaction(async (tx) => {
      await tx
        .update(schema.workspaces)
        .set({
          plan: "pro",
          features: {
            ...ctx.workspace.features,
            requestsQuota: Number.parseInt(price.metadata.quota_requests ?? "250000"),
          },
        })
        .where(eq(schema.workspaces.id, ctx.workspace.id));

      await insertAuditLogs(tx, ctx.workspace.auditLogBucket.id, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "workspace.update",
        description: `Changed plan to ${price.nickname}`,
        resources: [
          {
            type: "workspace",
            id: ctx.workspace.id,
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    });

    return { title: "Your plan has been changed" };
  });
