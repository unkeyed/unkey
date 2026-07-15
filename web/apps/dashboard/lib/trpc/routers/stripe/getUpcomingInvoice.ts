import { getStripeClient } from "@/lib/stripe";
import { deployBillingConfig } from "@/lib/stripe/deployBilling";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";

const upcomingInvoiceSchema = z
  .object({
    /** Billing period bounds, unix millis. */
    periodStart: z.number(),
    periodEnd: z.number(),
    /** Invoice total in the smallest currency unit (cents). */
    total: z.number(),
    currency: z.string(),
    /**
     * Sum of the Deploy metered line items (cents), or null when Deploy
     * billing is not configured. Plan fees are excluded; this is the
     * usage-so-far number the Deploy card plots against its credits.
     */
    deployUsageAmount: z.number().nullable(),
  })
  .nullable();

/**
 * Previews the workspace's upcoming Stripe invoice: the one headline number
 * (what the next bill will be), the billing period, and how much of it is
 * metered Deploy usage. Null when there is no subscription to preview.
 */
export const getUpcomingInvoice = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .output(upcomingInvoiceSchema)
  .query(async ({ ctx }) => {
    if (!ctx.workspace.stripeCustomerId || !ctx.workspace.stripeSubscriptionId) {
      return null;
    }

    const stripe = getStripeClient();
    const config = await deployBillingConfig();

    try {
      const invoice = await stripe.invoices.createPreview({
        customer: ctx.workspace.stripeCustomerId,
        subscription: ctx.workspace.stripeSubscriptionId,
      });

      // The first page covers our item count (one API plan, one Deploy
      // plan-fee, four meters); anything beyond it would only make the
      // usage split incomplete, never wrong, so we don't paginate.
      const deployUsageAmount = config
        ? invoice.lines.data
            .filter((line) => {
              const priceId = line.pricing?.price_details?.price;
              return typeof priceId === "string" && config.meteredPriceIds.includes(priceId);
            })
            .reduce((sum, line) => sum + line.amount, 0)
        : null;

      return {
        periodStart: invoice.period_start * 1000,
        periodEnd: invoice.period_end * 1000,
        total: invoice.total,
        currency: invoice.currency,
        deployUsageAmount,
      };
    } catch (err) {
      // Stripe throws when there is no upcoming invoice (e.g. the
      // subscription is fully canceled). That's an empty state, not an error.
      if (err instanceof Error && "code" in err && err.code === "invoice_upcoming_none") {
        return null;
      }
      throw err;
    }
  });
