import { z } from "zod";
import { getStripeClient } from "@/lib/stripe";
import { deployIncludedCreditCents } from "@/lib/stripe/deployCredits";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";

const creditSchema = z.object({
  /**
   * Included Deploy usage credit for the current period (cents), or null when
   * the workspace has no Stripe customer. The Deploy card nets month-to-date
   * usage against this to estimate the bill.
   */
  includedCreditCents: z.number().nullable(),
});

/**
 * Included credit changes about once a month (renewal) plus on plan changes, so
 * a short server-side cache keeps repeated dashboard loads off Stripe's
 * creditGrants API without letting a plan change read stale for long. Keyed per
 * customer; the process-local Map mirrors the resolution cache in deployBilling.
 */
const CACHE_TTL_MS = 60 * 1000;
const cache = new Map<string, { expiresAt: number; cents: number }>();

/**
 * The workspace's currently-included Deploy usage credit, read from Stripe's
 * credit grants and cached. See [[deployIncludedCreditCents]] for why this is
 * the granted amount (prorated on mid-cycle changes), not the catalog plan fee.
 */
export const getDeployCredit = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .output(creditSchema)
  .query(async ({ ctx }) => {
    const customerId = ctx.workspace.stripeCustomerId;
    if (!customerId) {
      return { includedCreditCents: null };
    }

    const now = Date.now();
    const hit = cache.get(customerId);
    if (hit && hit.expiresAt > now) {
      return { includedCreditCents: hit.cents };
    }

    const stripe = getStripeClient();
    const cents = await deployIncludedCreditCents(stripe, customerId);
    cache.set(customerId, { expiresAt: now + CACHE_TTL_MS, cents });
    return { includedCreditCents: cents };
  });
