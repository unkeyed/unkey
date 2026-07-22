import { getStripeClient } from "@/lib/stripe";
import {
  expandableId,
  retrieveWorkspaceCheckoutSession,
  throwMaskedStripeError,
} from "@/lib/trpc/routers/utils/stripe";
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
      // Needed post-checkout, before the workspace has a stripeCustomerId.
      // The session must be the workspace's and name this setup intent.
      // Empty is rejected rather than falling back to the workspace customer,
      // which would silently skip that check.
      sessionId: z.string().min(1, "Stripe checkout session ID is required").optional(),
    }),
  )
  .output(setupIntentSchema)
  .query(async ({ ctx, input }) => {
    const stripe = getStripeClient();

    let allowedCustomerId: string | null = null;

    if (input.sessionId) {
      const session = await retrieveWorkspaceCheckoutSession({
        stripe,
        sessionId: input.sessionId,
        workspaceId: ctx.workspace.id,
        notFoundMessage: "Setup intent not found",
      });

      // The session must also reference the requested setup intent, otherwise
      // an owned session could be used to read an unrelated one.
      if (expandableId(session.setup_intent) !== input.setupIntentId) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Setup intent not found",
        });
      }

      allowedCustomerId = expandableId(session.customer);
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

      if (expandableId(setupIntent.customer) !== allowedCustomerId) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Setup intent not found",
        });
      }

      return {
        id: setupIntent.id,
        client_secret: setupIntent.client_secret,
        payment_method: expandableId(setupIntent.payment_method),
        status: setupIntent.status,
        usage: setupIntent.usage,
      };
    } catch (error) {
      // If error is already a TRPCError, rethrow unchanged
      if (error instanceof TRPCError) {
        throw error;
      }

      // A nonexistent setup intent must be indistinguishable from a foreign
      // one, so a caller cannot probe which ids exist.
      if (error instanceof Stripe.errors.StripeError) {
        throwMaskedStripeError(error, "Setup intent not found");
      }

      // Handle unknown errors
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to retrieve setup intent",
        cause: error,
      });
    }
  });
