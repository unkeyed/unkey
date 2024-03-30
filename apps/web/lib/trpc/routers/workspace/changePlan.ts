import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { defaultProSubscriptions } from "@unkey/billing";
import Stripe from "stripe";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const changeWorkspacePlan = t.procedure
  .use(auth)
  .input(
    z.object({
      workspaceId: z.string(),
      plan: z.enum(["free", "pro"]),
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
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.id, input.workspaceId), isNull(table.deletedAt)),
    });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "workspace not found",
      });
    }
    if (workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({
        code: "UNAUTHORIZED",
        message: "you are not allowed to modify this workspace",
      });
    }
    const now = new Date();

    if (
      workspace.planChanged &&
      workspace.planChanged.getUTCFullYear() === now.getUTCFullYear() &&
      workspace.planChanged.getUTCMonth() === now.getUTCMonth()
    ) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message:
          "You have already changed your plan this month, please wait until next month or contact support@unkey.dev",
      });
    }

    if (workspace.plan === input.plan) {
      if (workspace.planDowngradeRequest) {
        // The user wants to resubscribe
        await db.transaction(async (tx) => {
          await tx
            .update(schema.workspaces)
            .set({
              planDowngradeRequest: null,
            })
            .where(eq(schema.workspaces.id, input.workspaceId));
          await ingestAuditLogs({
            workspaceId: workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "workspace.update",
            description: "Removed downgrade request",
            resources: [
              {
                type: "workspace",
                id: workspace.id,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          });
        });
        return {
          title: "You have resubscribed",
        };
      }
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "workspace already on this plan",
      });
    }

    switch (input.plan) {
      case "free": {
        // TODO: create invoice
        await db.transaction(async (tx) => {
          await tx
            .update(schema.workspaces)
            .set({
              planDowngradeRequest: "free",
            })
            .where(eq(schema.workspaces.id, input.workspaceId));
          await ingestAuditLogs({
            workspaceId: workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "workspace.update",
            description: "Requested downgrade to 'free'",
            resources: [
              {
                type: "workspace",
                id: workspace.id,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          });
        });
        return {
          title: "Your plan is scheduled to downgrade on the first of next month.",
          message:
            "You have access to all features until then and can reactivate your subscription at any point.",
        };
      }
      case "pro": {
        if (!workspace.stripeCustomerId) {
          throw new TRPCError({
            code: "PRECONDITION_FAILED",
            message: "Please add a payment method first",
          });
        }
        const paymentMethods = await stripe.customers.listPaymentMethods(
          workspace.stripeCustomerId,
        );
        if (!paymentMethods || paymentMethods.data.length === 0) {
          throw new TRPCError({
            code: "PRECONDITION_FAILED",
            message: "Please add a payment method first",
          });
        }
        await db.transaction(async (tx) => {
          await tx
            .update(schema.workspaces)
            .set({
              plan: "pro",
              planChanged: new Date(),
              subscriptions: defaultProSubscriptions(),
              planDowngradeRequest: null,
            })
            .where(eq(schema.workspaces.id, input.workspaceId));
          await ingestAuditLogs({
            workspaceId: workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "workspace.update",
            description: "Changed plan to 'pro'",
            resources: [
              {
                type: "workspace",
                id: workspace.id,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          });
        });
        return { title: "Your workspace has been upgraded" };
      }
    }
  });
