import { getStripeClient } from "@/lib/stripe";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";

const checkoutSessionSchema = z.object({
  id: z.string(),
  customer: z.string().nullable(),
  client_reference_id: z.string().nullable(),
  setup_intent: z.string().nullable(),
  payment_status: z.string(),
  status: z.string(),
});

export const getCheckoutSession = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      sessionId: z.string(),
    }),
  )
  .output(checkoutSessionSchema)
  .query(async ({ ctx, input }) => {
    const stripe = getStripeClient();

    try {
      const session = await stripe.checkout.sessions.retrieve(input.sessionId);

      if (!session) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Checkout session not found",
        });
      }

      // Sessions for billing are created with `client_reference_id` set to the
      // workspace id. Reject if the session was created for a different
      // workspace — otherwise an attacker can phish a victim into hitting
      // /success?session_id=<attacker_session> while logged in, leaking the
      // attacker's customer id to the victim's UI and triggering downstream
      // workspace mutations.
      if (session.client_reference_id !== ctx.workspace.id) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Checkout session not found",
        });
      }

      return {
        id: session.id,
        customer: session.customer ? session.customer.toString() : null,
        client_reference_id: session.client_reference_id,
        setup_intent: session.setup_intent ? session.setup_intent.toString() : null,
        payment_status: session.payment_status,
        status: session.status || "",
      };
    } catch (error) {
      if (error instanceof Stripe.errors.StripeError) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: `Stripe error: ${error.message}`,
        });
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to retrieve checkout session",
      });
    }
  });
