import { getStripeClient } from "@/lib/stripe";
import { handleStripeError } from "@/lib/trpc/routers/utils/stripe";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";

const customerSchema = z.object({
  id: z.string(),
  email: z.string().nullable(),
  name: z.string().nullable(),
  invoice_settings: z
    .object({
      default_payment_method: z.string().nullable(),
    })
    .nullable(),
});

export const getCustomer = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      customerId: z.string(),
    }),
  )
  .output(customerSchema)
  .query(async ({ input }) => {
    const stripe = getStripeClient();

    try {
      const customer = await stripe.customers.retrieve(input.customerId);

      if (customer.deleted) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Customer has been deleted",
        });
      }

      // Extract default payment method ID, handling both string and expanded object
      let defaultPaymentMethodId: string | null = null;
      if (customer.invoice_settings?.default_payment_method) {
        const paymentMethod = customer.invoice_settings.default_payment_method;
        if (typeof paymentMethod === "string") {
          defaultPaymentMethodId = paymentMethod;
        } else if (typeof paymentMethod === "object" && paymentMethod.id) {
          // Expanded PaymentMethod object
          defaultPaymentMethodId = paymentMethod.id;
        }
      }

      return {
        id: customer.id,
        email: customer.email,
        name: customer.name ?? null,
        invoice_settings: customer.invoice_settings
          ? {
              default_payment_method: defaultPaymentMethodId,
            }
          : null,
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
        message: "Failed to retrieve customer",
      });
    }
  });
