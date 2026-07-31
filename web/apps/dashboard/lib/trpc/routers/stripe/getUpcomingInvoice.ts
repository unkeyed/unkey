import { getStripeClient } from "@/lib/stripe";
import { deployBillingConfig } from "@/lib/stripe/deployBilling";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import type Stripe from "stripe";
import { z } from "zod";

const invoiceHalfSchema = z.object({
  /** Billing period bounds, unix millis. */
  periodStart: z.number(),
  periodEnd: z.number(),
  /** Invoice total in the smallest currency unit (cents). */
  total: z.number(),
  currency: z.string(),
});

const upcomingInvoiceSchema = z
  .object({
    /** The API product's upcoming invoice, or null when there is no API subscription. */
    api: invoiceHalfSchema.nullable(),
    /**
     * The Deploy product's upcoming invoice, or null when there is no Deploy
     * subscription. usageAmount is the sum of its metered line items (cents), or
     * null when Deploy billing is not configured; it is the usage-so-far number
     * the Deploy card plots against its credits.
     */
    deploy: invoiceHalfSchema.extend({ usageAmount: z.number().nullable() }).nullable(),
  })
  .nullable();

/**
 * Previews the workspace's upcoming Stripe invoices. Each product owns its own
 * subscription now, so this returns two independent previews: the API half and
 * the Deploy half (with its metered usage-so-far). Null when there is no
 * customer at all; each half is null when that product has no subscription.
 */
export const getUpcomingInvoice = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .output(upcomingInvoiceSchema)
  .query(async ({ ctx }) => {
    if (!ctx.workspace.stripeCustomerId) {
      return null;
    }
    const customerId = ctx.workspace.stripeCustomerId;

    const stripe = getStripeClient();

    // Stripe throws when there is no upcoming invoice (e.g. the subscription is
    // fully canceled). That's an empty state, not an error.
    const preview = async (subscriptionId: string): Promise<Stripe.Invoice | null> => {
      try {
        return await stripe.invoices.createPreview({
          customer: customerId,
          subscription: subscriptionId,
        });
      } catch (err) {
        if (err instanceof Error && "code" in err && err.code === "invoice_upcoming_none") {
          return null;
        }
        throw err;
      }
    };

    const [config, apiInvoice, deployInvoice] = await Promise.all([
      ctx.workspace.stripeDeploySubscriptionId ? deployBillingConfig() : null,
      ctx.workspace.stripeSubscriptionId ? preview(ctx.workspace.stripeSubscriptionId) : null,
      ctx.workspace.stripeDeploySubscriptionId
        ? preview(ctx.workspace.stripeDeploySubscriptionId)
        : null,
    ]);

    // The first page covers our per-product item count, so anything beyond it
    // would only make the usage split incomplete, never wrong; we don't paginate.
    const deployUsageAmount =
      deployInvoice && config
        ? deployInvoice.lines.data
            .filter((line) => {
              const priceId = line.pricing?.price_details?.price;
              return typeof priceId === "string" && config.meteredPriceIds.includes(priceId);
            })
            .reduce((sum, line) => sum + line.amount, 0)
        : null;

    return {
      api: apiInvoice
        ? {
            periodStart: apiInvoice.period_start * 1000,
            periodEnd: apiInvoice.period_end * 1000,
            total: apiInvoice.total,
            currency: apiInvoice.currency,
          }
        : null,
      deploy: deployInvoice
        ? {
            periodStart: deployInvoice.period_start * 1000,
            periodEnd: deployInvoice.period_end * 1000,
            total: deployInvoice.total,
            currency: deployInvoice.currency,
            usageAmount: deployUsageAmount,
          }
        : null,
    };
  });
