import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

export const updateWorkspaceStripeCustomer = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(
    z.object({
      sessionId: z.string().min(1, "Stripe checkout session ID is required"),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const stripe = getStripeClient();

    // Resolve customer id server-side from the checkout session, and verify
    // the session was created for this workspace. This prevents an attacker
    // from tricking a logged-in user into binding the attacker's Stripe
    // customer to the victim's workspace via a /success?session_id=... link.
    const session = await stripe.checkout.sessions.retrieve(input.sessionId);
    if (!session) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Checkout session not found",
      });
    }
    if (session.client_reference_id !== ctx.workspace.id) {
      throw new TRPCError({
        code: "FORBIDDEN",
        message: "Checkout session does not belong to this workspace",
      });
    }

    const stripeCustomerId =
      typeof session.customer === "string" ? session.customer : (session.customer?.id ?? null);
    if (!stripeCustomerId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Checkout session does not have a customer",
      });
    }

    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.workspaces)
          .set({
            stripeCustomerId,
          })
          .where(eq(schema.workspaces.id, ctx.workspace.id));

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "workspace.update",
          description: "Updated Stripe customer ID",
          resources: [
            {
              type: "workspace",
              id: ctx.workspace.id,
              name: ctx.workspace.name,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update the workspace Stripe customer. Please try again or contact support@unkey.com",
        });
      });

    return {
      success: true,
    };
  });
