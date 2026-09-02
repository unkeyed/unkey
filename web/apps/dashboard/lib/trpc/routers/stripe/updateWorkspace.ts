import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { insertAuditLogs } from "@/lib/audit";
import { db, schema, transactionWithRetry } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";
import { expandableId, retrieveCompletedWorkspaceCheckoutSession } from "../utils/stripe";

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
    const session = await retrieveCompletedWorkspaceCheckoutSession({
      stripe,
      sessionId: input.sessionId,
      workspaceId: ctx.workspace.id,
      notFoundMessage: "Checkout session not found for this workspace",
    });

    const stripeCustomerId = expandableId(session.customer);
    if (!stripeCustomerId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Checkout session does not have a customer",
      });
    }

    try {
      await transactionWithRetry(db, async (tx) => {
        // Upsert: binding the Stripe customer is the first billing write a
        // workspace gets, so create the billing row if none exists yet (a
        // workspace created before it had one). On conflict only the customer
        // id is set, leaving tier and the rest of the row untouched.
        await tx
          .insert(schema.workspaceBilling)
          .values({
            workspaceId: ctx.workspace.id,
            stripeCustomerId,
          })
          .onDuplicateKeyUpdate({ set: { stripeCustomerId } });

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
      });
    } catch (err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        // No "contact support" line: /success appends one when it renders.
        message: "We are unable to update the workspace Stripe customer. Please try again.",
        cause: err,
      });
    }

    return {
      success: true,
    };
  });
