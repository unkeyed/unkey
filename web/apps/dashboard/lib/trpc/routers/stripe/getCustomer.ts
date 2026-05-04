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
      // Either an already-bound workspace customer id, or a checkout session
      // id whose `client_reference_id` matches this workspace. Customer ids
      // are not secret, so we never trust a raw input — we always verify the
      // customer is one this workspace is allowed to read.
      customerId: z.string().optional(),
      sessionId: z.string().optional(),
    }),
  )
  .output(customerSchema)
  .query(async ({ ctx, input }) => {
    const stripe = getStripeClient();

    let resolvedCustomerId: string | null = null;

    if (input.sessionId) {
      const session = await stripe.checkout.sessions.retrieve(input.sessionId);
      if (!session || session.client_reference_id !== ctx.workspace.id) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Customer not found",
        });
      }
      resolvedCustomerId =
        typeof session.customer === "string"
          ? session.customer
          : (session.customer?.id ?? null);
    } else if (input.customerId && ctx.workspace.stripeCustomerId) {
      if (input.customerId !== ctx.workspace.stripeCustomerId) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Customer not found",
        });
      }
      resolvedCustomerId = input.customerId;
    }

    if (!resolvedCustomerId) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Customer not found",
      });
    }

    try {
      const customer = await stripe.customers.retrieve(resolvedCustomerId);

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
