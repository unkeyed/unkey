import { getStripeClient } from "@/lib/stripe";
import { handleStripeError } from "@/lib/trpc/routers/utils/stripe";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";

const updateCustomerInputSchema = z.object({
  customerId: z.string(),
  paymentMethod: z.string(),
});

const customerSchema = z.object({
  id: z.string(),
});

export const updateCustomer = workspaceProcedure
  .use(withRatelimit(ratelimit.update))
  .input(updateCustomerInputSchema)
  .output(customerSchema)
  .mutation(async ({ ctx, input }) => {
    const stripe = getStripeClient();

    // Only allow updating the customer recorded on the caller's own workspace.
    // Without this check a tenant could chain getCheckoutSession leaks into
    // changing another workspace's default payment method.
    if (
      !ctx.workspace.stripeCustomerId ||
      ctx.workspace.stripeCustomerId !== input.customerId
    ) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Customer not found",
      });
    }

    try {
      const customer = await stripe.customers.update(input.customerId, {
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
      });
    }
  });
