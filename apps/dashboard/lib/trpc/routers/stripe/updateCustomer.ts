import { stripeEnv } from "@/lib/env";
import { ratelimit, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
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

export const updateCustomer = t.procedure
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.update))
  .input(updateCustomerInputSchema)
  .output(customerSchema)
  .mutation(async ({ input }) => {
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
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: `Stripe error: ${error.message}`,
        });
      }

      // Handle unknown errors
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to update customer",
      });
    }
  });
