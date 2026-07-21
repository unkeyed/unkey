import { Stripe } from "stripe";

/**
 * Aligns every active Stripe subscription's billing cycle anchor to the 1st of
 * the month at 00:00:00 UTC, matching what new subscriptions get via
 * `billing_cycle_anchor_config: { day_of_month: 1, hour: 0, minute: 0, second: 0 }`
 * (see web/apps/dashboard/lib/trpc/routers/stripe/createSubscription.ts).
 *
 * Stripe does not allow changing billing_cycle_anchor_config on an existing
 * subscription; the documented way to move the anchor is to set `trial_end` to
 * the desired timestamp: "The billing_cycle_anchor will be updated to the
 * trial_end value." So for each misaligned subscription this script sets
 * trial_end to the next 1st-of-month 00:00:00 UTC with proration disabled.
 *
 * Side effects to be aware of before running with --apply:
 * - The subscription's status becomes "trialing" until the 1st. Our code
 *   treats trialing as subscribed (subscribeDeploy/createSubscription accept
 *   active and trialing), but anything that special-cases "active" will see a
 *   temporary status change.
 * - The current billing period ends at the new trial_end: metered usage
 *   accrued so far is invoiced on the 1st together with the next plan fee.
 * - proration_behavior "none" means no credit for the remainder of an already
 *   paid plan fee; the customer effectively gets the stub period included.
 *
 * Usage (from web/tools/migrate):
 *   dotenv -e .env -- npx tsx ./align_billing_cycle_anchor.ts          # dry run
 *   dotenv -e .env -- npx tsx ./align_billing_cycle_anchor.ts --apply  # mutate
 */
async function main() {
  const apiKey = process.env.STRIPE_SECRET_KEY;
  if (!apiKey) {
    throw new Error("STRIPE_SECRET_KEY is not set");
  }
  const apply = process.argv.includes("--apply");

  const stripe = new Stripe(apiKey, {
    apiVersion: "2023-10-16",
    typescript: true,
  });

  let total = 0;
  let aligned = 0;
  let skipped = 0;
  let updated = 0;

  for await (const sub of stripe.subscriptions.list({ status: "active", limit: 100 })) {
    total++;

    const anchor = new Date(sub.billing_cycle_anchor * 1000);
    const isAligned =
      anchor.getUTCDate() === 1 &&
      anchor.getUTCHours() === 0 &&
      anchor.getUTCMinutes() === 0 &&
      anchor.getUTCSeconds() === 0;

    if (isAligned) {
      aligned++;
      continue;
    }

    // Moving the anchor on a subscription that is winding down would shift its
    // end date; leave those alone and let the cancellation play out.
    if (sub.cancel_at_period_end || sub.cancel_at !== null) {
      skipped++;
      console.log(
        `skip ${sub.id} (customer ${sub.customer}): cancelling, anchor ${anchor.toISOString()}`,
      );
      continue;
    }

    const trialEnd = nextFirstOfMonthUTC(new Date());
    console.log(
      `${apply ? "update" : "would update"} ${sub.id} (customer ${sub.customer}): ` +
        `anchor ${anchor.toISOString()} -> ${trialEnd.toISOString()}`,
    );

    if (apply) {
      await stripe.subscriptions.update(
        sub.id,
        {
          trial_end: Math.floor(trialEnd.getTime() / 1000),
          proration_behavior: "none",
        },
        { idempotencyKey: `align-anchor:${sub.id}:${trialEnd.toISOString()}` },
      );
      updated++;
    }
  }

  console.log(
    `done: ${total} active subscriptions, ${aligned} already aligned, ` +
      `${skipped} skipped (cancelling), ${apply ? `${updated} updated` : "dry run (pass --apply to mutate)"}`,
  );
}

/**
 * First day of the following month at 00:00:00 UTC, strictly in the future so
 * Stripe accepts it as a trial_end. Running exactly at a month boundary rolls
 * to the month after, which only delays alignment by one cycle.
 */
function nextFirstOfMonthUTC(now: Date): Date {
  return new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 1, 0, 0, 0));
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
