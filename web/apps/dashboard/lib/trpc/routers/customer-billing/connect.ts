import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
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

// Disconnect Stripe account - uses the API route for programmatic access
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
