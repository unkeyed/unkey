import { Workspace, db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { clerkClient } from "@clerk/nextjs";
import { TRPCError } from "@trpc/server";
import { defaultProSubscriptions } from "@unkey/billing";
import { newId } from "@unkey/id";
import Stripe from "stripe";
import { z } from "zod";
import { auth, t } from "../trpc";

export const workspaceRouter = t.router({
  create: t.procedure
    .use(auth)
    .input(
      z.object({
        name: z.string().min(1).max(50),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const userId = ctx.user?.id;
      if (!userId) {
        throw new TRPCError({ code: "UNAUTHORIZED", message: "unable to find userId" });
      }

      const org = await clerkClient.organizations.createOrganization({
        name: input.name,
        createdBy: userId,
      });

      const workspace: Workspace = {
        id: newId("workspace"),
        tenantId: org.id,
        name: input.name,
        plan: "pro",
        stripeCustomerId: null,
        stripeSubscriptionId: null,
        trialEnds: new Date(Date.now() + 1000 * 60 * 60 * 24 * 14), // 2 weeks
        features: {},
        betaFeatures: {},
        planLockedUntil: null,
        planChanged: null,
        subscriptions: defaultProSubscriptions(),
        createdAt: new Date(),
        deletedAt: null,
      };
      await db.transaction(async (tx) => {
        await tx.insert(schema.workspaces).values(workspace);
        await tx.insert(schema.auditLogs).values({
          id: newId("auditLog"),
          time: new Date(),
          workspaceId: workspace.id,
          actorType: "user",
          actorId: ctx.user.id,
          event: "workspace.create",
          description: `Workspace ${input.name} created`,
        });
      });

      return {
        workspace,
        organizationId: org.id,
      };
    }),

  changePlan: t.procedure
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
        throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: "stripe env not set" });
      }
      const stripe = new Stripe(env.STRIPE_SECRET_KEY, {
        apiVersion: "2022-11-15",
        typescript: true,
      });
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.id, input.workspaceId), isNull(table.deletedAt)),
      });

      if (!workspace) {
        throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
      }
      if (workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({
          code: "UNAUTHORIZED",
          message: "you are not allowed to modify this workspace",
        });
      }

      if (
        workspace.planChanged &&
        Date.now() < workspace.planChanged.getTime() + 1000 * 60 * 60 * 24
      ) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message:
            "You can not change your plan multiple times per day, please wait 24h or contact support@unkey.dev",
        });
      }

      if (workspace.plan === input.plan) {
        throw new TRPCError({ code: "BAD_REQUEST", message: "workspace already on this plan" });
      }

      switch (input.plan) {
        case "free": {
          // TODO: create invoice
          await db.transaction(async (tx) => {
            await tx
              .update(schema.workspaces)
              .set({
                plan: "free",
                planChanged: new Date(),
                subscriptions: null,
              })
              .where(eq(schema.workspaces.id, input.workspaceId));
            await tx.insert(schema.auditLogs).values({
              id: newId("auditLog"),
              time: new Date(),
              workspaceId: workspace.id,
              actorType: "user",
              actorId: ctx.user.id,
              event: "workspace.update",
              description: `Workspace ${workspace.name} changed plan to free`,
            });
          });
          break;
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
              })
              .where(eq(schema.workspaces.id, input.workspaceId));
            await tx.insert(schema.auditLogs).values({
              id: newId("auditLog"),
              time: new Date(),
              workspaceId: workspace.id,
              actorType: "user",
              actorId: ctx.user.id,
              event: "workspace.update",
              description: `Workspace ${workspace.name} changed plan to pro`,
            });
          });
          break;
        }
      }
    }),
});
