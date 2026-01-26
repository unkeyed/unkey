import { auth } from "@/lib/auth/server";
import { db } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { syncCanceledSubscription } from "@/lib/stripe/sync";

export const runtime = "nodejs";

const GRACE_PERIOD_MS = 7 * 24 * 60 * 60 * 1000; // 7 days

export const GET = async (req: Request): Promise<Response> => {
  const authHeader = req.headers.get("authorization");
  const cronSecret = process.env.CRON_SECRET;

  if (cronSecret && authHeader !== `Bearer ${cronSecret}`) {
    return new Response("Unauthorized", { status: 401 });
  }

  if (!cronSecret && process.env.NODE_ENV === "production") {
    console.error("CRON_SECRET is not configured in production");
    return new Response("Server configuration error", { status: 500 });
  }

  try {
    const stripe = getStripeClient();
    const now = Date.now();
    const gracePeriodExpiry = now - GRACE_PERIOD_MS;

    const expiredWorkspaces = await db.query.workspaces.findMany({
      where: (table, { and, or, lt, isNotNull, isNull, inArray }) =>
        and(
          or(inArray(table.subscriptionStatus, ["past_due", "unpaid"])),
          isNotNull(table.paymentFailedAt),
          lt(table.paymentFailedAt, gracePeriodExpiry),
          isNull(table.deletedAtM),
        ),
    });

    const results = [];

    for (const workspace of expiredWorkspaces) {
      try {
        // Step 1: Deactivate memberships first (idempotent operation)
        // This ensures members lose access before any billing changes
        const memberships = await auth.getOrganizationMemberList(workspace.orgId);

        if (memberships.data.length > 1) {
          const sortedMembers = memberships.data.sort((a, b) => {
            if (a.role === "admin" && b.role !== "admin") return -1;
            if (b.role === "admin" && a.role !== "admin") return 1;
            return new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime();
          });

          const keepMemberId = sortedMembers[0].id;
          const deactivationErrors = [];

          for (const member of sortedMembers.slice(1)) {
            if (member.id === keepMemberId) continue;

            try {
              // deactivateMembership is idempotent - safe to retry
              await auth.deactivateMembership(member.id);
            } catch (error) {
              console.error(
                `Failed to deactivate membership ${member.id} for workspace ${workspace.id}:`,
                error,
              );
              deactivationErrors.push({
                memberId: member.id,
                error: error instanceof Error ? error.message : "Unknown error",
              });
            }
          }

          // If any deactivations failed, throw to prevent subscription cancellation
          if (deactivationErrors.length > 0) {
            throw new Error(
              `Failed to deactivate ${deactivationErrors.length} member(s): ${JSON.stringify(deactivationErrors)}`,
            );
          }
        }

        // Step 2: Cancel Stripe subscription (only after members are deactivated)
        if (workspace.stripeSubscriptionId) {
          try {
            await stripe.subscriptions.cancel(workspace.stripeSubscriptionId);
          } catch (error) {
            // If subscription is already canceled, continue (idempotent)
            if (
              error instanceof Error &&
              !error.message.includes("No such subscription") &&
              !error.message.includes("already canceled")
            ) {
              throw error;
            }
          }
        }

        // Step 3: Sync canceled subscription to database (final step)
        await syncCanceledSubscription(workspace.id);

        results.push({
          workspaceId: workspace.id,
          status: "success",
        });
      } catch (error) {
        console.error(`Failed to expire grace period for workspace ${workspace.id}:`, error);
        results.push({
          workspaceId: workspace.id,
          status: "error",
          error: error instanceof Error ? error.message : "Unknown error",
        });
      }
    }

    return new Response(
      JSON.stringify({
        processed: results.length,
        results,
      }),
      {
        status: 200,
        headers: { "Content-Type": "application/json" },
      },
    );
  } catch (error) {
    console.error("Grace period expiration cron error:", error);
    return new Response(
      JSON.stringify({
        error: error instanceof Error ? error.message : "Unknown error",
      }),
      {
        status: 500,
        headers: { "Content-Type": "application/json" },
      },
    );
  }
};
