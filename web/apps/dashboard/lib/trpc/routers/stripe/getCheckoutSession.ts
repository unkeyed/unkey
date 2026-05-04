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

      // The dashboard sets client_reference_id to the workspace id when
      // creating the checkout session. Reject any session that does not
      // belong to the caller's workspace; cs_* ids leak through success_url
      // query params, browser history, referer headers, etc.
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
