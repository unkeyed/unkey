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

      if (!customer || customer.deleted) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Customer not found or has been deleted",
        });
      }

      return {
        id: customer.id,
        email: customer.email,
        name: customer.name ?? null,
        invoice_settings: customer.invoice_settings
          ? {
              default_payment_method: customer.invoice_settings.default_payment_method
                ? customer.invoice_settings.default_payment_method.toString()
                : null,
            }
          : null,
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
        message: "Failed to retrieve customer",
      });
    }
  });
