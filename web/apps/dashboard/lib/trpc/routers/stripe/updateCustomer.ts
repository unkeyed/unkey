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
  // Either the workspace's bound customer id, or a checkout session whose
  // client_reference_id matches. During initial card-vault setup no
  // customer is bound yet, so callers pass sessionId and ownership is
  // verified from the session.
  customerId: z.string().optional(),
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

    // Resolve and verify ownership server-side; never trust a caller-supplied id.
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
        typeof session.customer === "string" ? session.customer : (session.customer?.id ?? null);
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
      const customer = await stripe.customers.update(resolvedCustomerId, {
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
