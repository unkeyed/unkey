import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { auth, t } from "../../trpc";
export const cancelSubscription = t.procedure.use(auth).mutation(async ({ ctx }) => {
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

  await stripe.subscriptions.cancel(ctx.workspace.stripeSubscriptionId, {
    prorate: true,
  });

  await db
    .update(schema.workspaces)
    .set({
      stripeSubscriptionId: null,
    })
    .where(eq(schema.workspaces.id, ctx.workspace.id));
  await db
    .insert(schema.quotas)
    .values({
      workspaceId: ctx.workspace.id,
      requestsPerMonth: 250_000,
      logsRetentionDays: 7,
      auditLogsRetentionDays: 30,
      team: false,
    })
    .onDuplicateKeyUpdate({
      set: {
        requestsPerMonth: 250_000,
        logsRetentionDays: 7,
        auditLogsRetentionDays: 30,
        team: false,
      },
    });

  await insertAuditLogs(db, ctx.workspace.auditLogBucket.id, {
    workspaceId: ctx.workspace.id,
    actor: {
      type: "user",
      id: ctx.user.id,
    },
    event: "workspace.update",
    description: "Cancelled subscription.",
    resources: [],
    context: {
      location: ctx.audit.location,
      userAgent: ctx.audit.userAgent,
    },
  });
});
