import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { getStripeClient } from "@/lib/stripe";
import {
  expandableId,
  retrieveWorkspaceCheckoutSession,
  throwMaskedStripeError,
} from "@/lib/trpc/routers/utils/stripe";
import {
  ratelimit,
  requireWorkspaceAdmin,
  withRatelimit,
  workspaceProcedure,
} from "@/lib/trpc/trpc";

const NOT_FOUND_MESSAGE = "Setup intent not found";

// `client_secret` is omitted: with Stripe.js it can attach or replace a payment
// method on the workspace's customer, and no caller needs it.
const setupIntentSchema = z.object({
  id: z.string(),
  payment_method: z.string().nullable(),
  status: z.string(),
  usage: z.string(),
});

export const getSetupIntent = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      setupIntentId: z.string(),
      // Post-checkout the workspace has no stripeCustomerId yet, so the session
      // is used to authorize instead. Optional, but rejected when empty.
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
        notFoundMessage: NOT_FOUND_MESSAGE,
      });

      if (expandableId(session.setup_intent) !== input.setupIntentId) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: NOT_FOUND_MESSAGE,
        });
      }

      allowedCustomerId = expandableId(session.customer);
    } else if (ctx.workspace.stripeCustomerId) {
      allowedCustomerId = ctx.workspace.stripeCustomerId;
    }

    if (!allowedCustomerId) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: NOT_FOUND_MESSAGE,
      });
    }

    try {
      const setupIntent = await stripe.setupIntents.retrieve(input.setupIntentId);

      const setupIntentCustomerId = expandableId(setupIntent.customer);

      if (setupIntentCustomerId !== allowedCustomerId) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: NOT_FOUND_MESSAGE,
        });
      }

      return {
        id: setupIntent.id,
        payment_method: expandableId(setupIntent.payment_method),
        status: setupIntent.status,
        usage: setupIntent.usage,
      };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      if (error instanceof Stripe.errors.StripeError) {
        throwMaskedStripeError(error, NOT_FOUND_MESSAGE);
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to retrieve setup intent",
        cause: error,
      });
    }
  });
