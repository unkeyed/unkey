import { getStripeClient } from "@/lib/stripe";
import {
  expandableId,
  handleStripeError,
  retrieveWorkspaceCheckoutSession,
} from "@/lib/trpc/routers/utils/stripe";
import {
  ratelimit,
  requireWorkspaceAdmin,
  withRatelimit,
  workspaceProcedure,
} from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";

const updateCustomerInputSchema = z.object({
  // The customer id is never taken from the client: it is resolved from the
  // verified session, or from the workspace when no sessionId is given.
  sessionId: z.string().optional(),
  paymentMethod: z.string(),
});

const customerSchema = z.object({
  id: z.string(),
});

export const updateCustomer = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .use(withRatelimit(ratelimit.update))
  .input(updateCustomerInputSchema)
  .output(customerSchema)
  .mutation(async ({ ctx, input }) => {
    const stripe = getStripeClient();

    // The session's customer is authoritative: setup-mode checkout always
    // creates a new customer and attaches the payment method to it, so
    // targeting the previously bound customer would fail.
    let customerId: string | null;

    if (input.sessionId) {
      const session = await retrieveWorkspaceCheckoutSession({
        stripe,
        sessionId: input.sessionId,
        workspaceId: ctx.workspace.id,
        notFoundMessage: "Customer not found",
      });
      customerId = expandableId(session.customer);
    } else {
      customerId = ctx.workspace.stripeCustomerId;
    }

    if (!customerId) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Customer not found",
      });
    }

    try {
      const customer = await stripe.customers.update(customerId, {
        invoice_settings: {
          default_payment_method: input.paymentMethod,
        },
      });

      if (!customer) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Customer not found or has been deleted",
        });
      }

      return {
        id: customer.id,
      };
    } catch (error) {
      // If error is already a TRPCError, rethrow unchanged
      if (error instanceof TRPCError) {
        throw error;
      }

      // Handle Stripe errors
      if (error instanceof Stripe.errors.StripeError) {
        handleStripeError(error);
      }

      // Handle unknown errors
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to update customer",
        cause: error,
      });
    }
  });
