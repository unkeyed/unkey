import { stripeEnv } from "@/lib/env";
import { ratelimit, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
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

export const getCheckoutSession = t.procedure
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      sessionId: z.string(),
    }),
  )
  .output(checkoutSessionSchema)
  .query(async ({ input }) => {
    const e = stripeEnv();
    if (!e) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Stripe is not configured",
      });
    }

    const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
      apiVersion: "2023-10-16",
      typescript: true,
    });

    try {
      const session = await stripe.checkout.sessions.retrieve(input.sessionId);

      if (!session) {
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
