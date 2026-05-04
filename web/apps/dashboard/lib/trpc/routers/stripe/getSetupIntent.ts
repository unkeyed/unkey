import { getStripeClient } from "@/lib/stripe";
import { handleStripeError } from "@/lib/trpc/routers/utils/stripe";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
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

export const getSetupIntent = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      setupIntentId: z.string(),
      // Optional sessionId path: needed for the post-checkout flow where the
      // workspace doesn't yet have a stripeCustomerId. The session must
      // belong to this workspace and reference the same setup intent.
      sessionId: z.string().optional(),
    }),
  )
  .output(setupIntentSchema)
  .query(async ({ ctx, input }) => {
    const stripe = getStripeClient();

    let allowedCustomerId: string | null = null;

    if (input.sessionId) {
      const session = await stripe.checkout.sessions.retrieve(input.sessionId);
      if (
        !session ||
        session.client_reference_id !== ctx.workspace.id ||
        (typeof session.setup_intent === "string"
          ? session.setup_intent
          : session.setup_intent?.id) !== input.setupIntentId
      ) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Setup intent not found",
        });
      }
      allowedCustomerId =
        typeof session.customer === "string"
          ? session.customer
          : (session.customer?.id ?? null);
    } else if (ctx.workspace.stripeCustomerId) {
      allowedCustomerId = ctx.workspace.stripeCustomerId;
    }

    if (!allowedCustomerId) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Setup intent not found",
      });
    }

    try {
      const setupIntent = await stripe.setupIntents.retrieve(input.setupIntentId);

      const setupIntentCustomerId =
        typeof setupIntent.customer === "string"
          ? setupIntent.customer
          : (setupIntent.customer?.id ?? null);

      if (setupIntentCustomerId !== allowedCustomerId) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Setup intent not found",
        });
      }

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
