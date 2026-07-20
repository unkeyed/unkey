import { getStripeClient } from "@/lib/stripe";
import { handleStripeError } from "@/lib/trpc/routers/utils/stripe";
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
  // Optional sessionId path: needed for the post-checkout flow where the
  // workspace doesn't yet have a stripeCustomerId. The session must belong to
  // this workspace. The customer id is never taken from the client.
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

    let customerId: string | null = ctx.workspace.stripeCustomerId;

    if (!customerId && input.sessionId) {
      const session = await stripe.checkout.sessions.retrieve(input.sessionId);
      if (!session || session.client_reference_id !== ctx.workspace.id) {
        throw new TRPCError({
          code: "FORBIDDEN",
          message: "Checkout session does not belong to this workspace",
        });
      }
      customerId =
        typeof session.customer === "string" ? session.customer : (session.customer?.id ?? null);
    }

    if (!customerId) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Customer not found or has been deleted",
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
      });
    }
  });
