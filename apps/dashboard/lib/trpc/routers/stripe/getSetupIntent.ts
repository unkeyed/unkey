import { getStripeClient } from "@/lib/stripe";
import { handleStripeError } from "@/lib/trpc/routers/utils/stripe";
import { ratelimit, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";

const setupIntentSchema = z.object({
  id: z.string(),
  client_secret: z.string().nullable(),
  payment_method: z.string().nullable(),
  status: z.string(),
  usage: z.string(),
});

export const getSetupIntent = t.procedure
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      setupIntentId: z.string(),
    }),
  )
  .output(setupIntentSchema)
  .query(async ({ input }) => {
    const stripe = getStripeClient();

    try {
      const setupIntent = await stripe.setupIntents.retrieve(input.setupIntentId);

      // Extract payment method ID, handling both string and expanded object
      let paymentMethodId: string | null = null;
      if (setupIntent.payment_method) {
        if (typeof setupIntent.payment_method === "string") {
          paymentMethodId = setupIntent.payment_method;
        } else if (
          typeof setupIntent.payment_method === "object" &&
          setupIntent.payment_method.id
        ) {
          // Expanded PaymentMethod object
          paymentMethodId = setupIntent.payment_method.id;
        }
      }

      return {
        id: setupIntent.id,
        client_secret: setupIntent.client_secret,
        payment_method: paymentMethodId,
        status: setupIntent.status,
        usage: setupIntent.usage,
      };
    } catch (error) {
      // If error is already a TRPCError, rethrow unchanged
      if (error instanceof TRPCError) {
        throw error;
      }

      // Map Stripe errors to appropriate TRPC error codes
      if (error instanceof Stripe.errors.StripeError) {
        handleStripeError(error);
      }

      // Handle unknown errors
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to retrieve setup intent",
      });
    }
  });
