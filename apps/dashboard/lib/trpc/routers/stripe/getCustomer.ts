import { stripeEnv } from "@/lib/env";
import { ratelimit, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
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

export const getCustomer = t.procedure
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      customerId: z.string(),
    }),
  )
  .output(customerSchema)
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
        message: "Failed to retrieve customer",
      });
    }
  });
