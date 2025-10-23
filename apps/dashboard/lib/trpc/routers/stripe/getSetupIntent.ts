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
    })
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

      if (!setupIntent) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Setup intent not found",
        });
      }

      return {
        id: setupIntent.id,
        client_secret: setupIntent.client_secret,
        payment_method: setupIntent.payment_method
          ? setupIntent.payment_method.toString()
          : null,
        status: setupIntent.status,
        usage: setupIntent.usage,
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
        message: "Failed to retrieve setup intent",
      });
    }
  });
