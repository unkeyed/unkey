import { getStripeClient } from "@/lib/stripe";
import {
  expandableId,
  retrieveCompletedWorkspaceCheckoutSession,
  throwRedactedStripeError,
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
  // No customer id from the client: it is derived from this session, which is
  // verified to belong to the workspace.
  sessionId: z.string().min(1, "Stripe checkout session ID is required"),
  // Stripe reads an empty string as "unset", which clears the card on file and
  // reports success.
  paymentMethod: z.string().min(1, "Payment method is required"),
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

    // The session's customer, not the workspace's: setup-mode checkout creates
    // a new customer and attaches the payment method to that one.
    const session = await retrieveCompletedWorkspaceCheckoutSession({
      stripe,
      sessionId: input.sessionId,
      workspaceId: ctx.workspace.id,
      notFoundMessage: "Customer not found",
    });
    const customerId = expandableId(session.customer);

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

      return {
        id: customer.id,
      };
    } catch (error) {
      if (error instanceof Stripe.errors.StripeError) {
        throwRedactedStripeError(error, "Failed to set the default payment method");
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to set the default payment method",
        cause: error,
      });
    }
  });
