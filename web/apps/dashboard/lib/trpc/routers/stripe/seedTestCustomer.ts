import { db, schema } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { TRPCError } from "@trpc/server";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

/**
 * Dev-only convenience: create a Stripe test customer for this workspace with
 * the signed-in user's email and Stripe's canned test Visa (`pm_card_visa`, the
 * 4242 4242 4242 4242 card), set it as the default payment method, and record
 * the customer id on workspace_billing. Saves re-entering sandbox card data on
 * every local run. Gated to non-production (independent of STRIPE_DEV_TEST_CLOCK,
 * which only controls invoice time-travel) so it can never run in prod, and
 * admin-only like the rest of billing.
 */
export const seedTestCustomer = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .mutation(async ({ ctx }) => {
    if (process.env.NODE_ENV === "production") {
      throw new TRPCError({
        code: "FORBIDDEN",
        message: "Stripe test seeding is only available outside production.",
      });
    }

    const stripe = getStripeClient();

    const customer = await stripe.customers.create({
      email: ctx.user.profile?.email ?? undefined,
      metadata: { workspace_id: ctx.workspace.id, seeded: "dev" },
    });

    // pm_card_visa is Stripe's shared test PaymentMethod for 4242 4242 4242 4242.
    const paymentMethod = await stripe.paymentMethods.attach("pm_card_visa", {
      customer: customer.id,
    });
    await stripe.customers.update(customer.id, {
      invoice_settings: { default_payment_method: paymentMethod.id },
    });

    // Upsert: the billing row may not exist yet for a freshly seeded workspace.
    await db
      .insert(schema.workspaceBilling)
      .values({ workspaceId: ctx.workspace.id, stripeCustomerId: customer.id })
      .onDuplicateKeyUpdate({ set: { stripeCustomerId: customer.id } });

    return { customerId: customer.id };
  });
