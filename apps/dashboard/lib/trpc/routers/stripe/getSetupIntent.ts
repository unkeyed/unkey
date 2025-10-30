import { stripeEnv } from "@/lib/env";
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
        const stripeError = error;
        let code: TRPCError["code"];

        // Map Stripe error types to TRPC error codes
        if (error instanceof Stripe.errors.StripeAuthenticationError) {
          code = "UNAUTHORIZED";
        } else if (error instanceof Stripe.errors.StripeRateLimitError) {
          code = "TOO_MANY_REQUESTS";
        } else if (error instanceof Stripe.errors.StripeInvalidRequestError) {
          code = "BAD_REQUEST";
        } else if (error instanceof Stripe.errors.StripePermissionError) {
          code = "FORBIDDEN";
        } else if (stripeError.statusCode === 404 || stripeError.code === "resource_missing") {
          code = "NOT_FOUND";
        } else if (error instanceof Stripe.errors.StripeAPIError) {
          code = "INTERNAL_SERVER_ERROR";
        } else if (error instanceof Stripe.errors.StripeConnectionError) {
          code = "INTERNAL_SERVER_ERROR";
        } else {
          // Default for other Stripe errors
          code = "INTERNAL_SERVER_ERROR";
        }

        throw new TRPCError({
          code,
          message: `Stripe error: ${stripeError.message}`,
        });
      }

      // Handle unknown errors
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to retrieve setup intent",
      });
    }
  });
