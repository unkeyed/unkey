import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

// Get connected account status
export const getConnectedAccount = workspaceProcedure.query(async ({ ctx }) => {
  const account = await db.query.stripeConnectedAccounts.findFirst({
    where: eq(schema.stripeConnectedAccounts.workspaceId, ctx.workspace.id),
  });

  if (!account || account.disconnectedAt) {
    return null;
  }

  return {
    id: account.id,
    workspaceId: account.workspaceId,
    stripeAccountId: account.stripeAccountId,
    scope: account.scope,
    connectedAt: account.connectedAt,
    disconnectedAt: account.disconnectedAt,
  };
});

// Get authorization URL for Stripe Connect OAuth
export const getAuthorizationUrl = workspaceProcedure
  .input(
    z.object({
      redirectUri: z.string().url(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    // Check if workspace has billing beta access
    if (!ctx.workspace.betaFeatures.billing) {
      throw new TRPCError({
        code: "FORBIDDEN",
        message: "Billing feature is not enabled for this workspace",
      });
    }

    const clientId = process.env.STRIPE_CONNECT_CLIENT_ID;
    if (!clientId) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Stripe Connect is not configured",
      });
    }

    const state = Buffer.from(
      JSON.stringify({
        workspaceId: ctx.workspace.id,
        timestamp: Date.now(),
      }),
    ).toString("base64");

    const params = new URLSearchParams({
      response_type: "code",
      client_id: clientId,
      scope: "read_write",
      redirect_uri: input.redirectUri,
      state,
    });

    return {
      authorizationUrl: `https://connect.stripe.com/oauth/authorize?${params.toString()}`,
    };
  });

// Disconnect Stripe account
export const disconnectAccount = workspaceProcedure.mutation(async ({ ctx }) => {
  const account = await db.query.stripeConnectedAccounts.findFirst({
    where: eq(schema.stripeConnectedAccounts.workspaceId, ctx.workspace.id),
  });

  if (!account) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "No connected Stripe account found",
    });
  }

  // Check if there are active end users
  const endUsers = await db.query.billingEndUsers.findMany({
    where: eq(schema.billingEndUsers.workspaceId, ctx.workspace.id),
  });

  if (endUsers.length > 0) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "Cannot disconnect account with active end users",
    });
  }

  await db
    .update(schema.stripeConnectedAccounts)
    .set({
      disconnectedAt: Date.now(),
      updatedAtM: Date.now(),
    })
    .where(eq(schema.stripeConnectedAccounts.id, account.id));

  await insertAuditLogs(db, {
    workspaceId: ctx.workspace.id,
    event: "stripeConnect.disconnect",
    actor: { type: "user", id: ctx.user.id },
    description: `Disconnected Stripe account ${account.stripeAccountId}`,
    resources: [
      {
        type: "stripeConnectedAccount",
        id: account.id,
        meta: { stripeAccountId: account.stripeAccountId },
      },
    ],
    context: { location: "", userAgent: undefined },
  });

  return { success: true };
});
