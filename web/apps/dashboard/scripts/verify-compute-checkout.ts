#!/usr/bin/env bun
/**
 * Verification harness for the no-card Compute checkout plan
 * (docs/plans/2026-07-02-001-fix-compute-checkout-subscription-mode-plan.md).
 *
 * It settles the plan's load-bearing Open Question — does a future day-1-anchored
 * subscription with proration_behavior "create_prorations" charge the prorated
 * partial period NOW, or $0 until the 1st? — against real Stripe under a test
 * clock, and prints a ready-to-open Checkout URL for the hosted-page steps that
 * cannot be driven headlessly. It reuses the app's own deployBillingConfig /
 * deployCheckoutLineItems so it exercises the real code path, not a copy.
 *
 * This is a manual verification aid, not a unit test: it needs a Stripe TEST
 * key and the Compute price lookup keys configured, so it is not part of the
 * automated suite. It is non-destructive — all state is created under a fresh
 * test clock and cleaned up at the end.
 *
 * Usage:
 *   STRIPE_SECRET_KEY=sk_test_... \
 *   STRIPE_LOOKUP_DEPLOY_STARTER=... STRIPE_LOOKUP_DEPLOY_PRO=... \
 *   STRIPE_LOOKUP_DEPLOY_BUSINESS=... STRIPE_LOOKUP_DEPLOY_METER_CPU=... \
 *   STRIPE_LOOKUP_DEPLOY_METER_MEMORY=... STRIPE_LOOKUP_DEPLOY_METER_EGRESS=... \
 *   STRIPE_LOOKUP_DEPLOY_METER_DISK=... \
 *   bun run scripts/verify-compute-checkout.ts --plan starter [--base-url http://localhost:3000]
 *
 * What it checks (plan scenario 6 core, headless):
 *   1. billing_cycle_anchor lands on the 1st of a month (anchor honored).
 *   2. Reports the first-invoice amount actually collected now (prorated vs $0).
 *   3. Confirms deployCheckoutLineItems shape is accepted by a real Checkout
 *      session and prints session.url + amount_total for the hosted-page steps
 *      (scenarios 1/2/6/7 end-to-end, which require a browser).
 */

import { parseArgs } from "node:util";
import { deployBillingConfig, deployCheckoutLineItems } from "@/lib/stripe/deployBilling";
import { DEPLOY_PLANS, type DeployPlan } from "@/lib/stripe/deployPlan";
import Stripe from "stripe";

function isDeployPlan(value: string): value is DeployPlan {
  return (DEPLOY_PLANS as readonly string[]).includes(value);
}

async function main() {
  const { values } = parseArgs({
    options: {
      plan: { type: "string", default: "starter" },
      "base-url": { type: "string", default: "http://localhost:3000" },
    },
  });

  const secretKey = process.env.STRIPE_SECRET_KEY;
  if (!secretKey) {
    throw new Error("STRIPE_SECRET_KEY is required.");
  }
  // Guard: never run this against a live account — it creates subscriptions.
  if (!secretKey.startsWith("sk_test_")) {
    throw new Error("Refusing to run: STRIPE_SECRET_KEY must be a TEST-mode key (sk_test_...).");
  }

  const plan = values.plan ?? "starter";
  if (!isDeployPlan(plan)) {
    throw new Error(`--plan must be one of ${DEPLOY_PLANS.join(", ")}`);
  }

  // Same API version the app pins (gates billing_cycle_anchor_config on Checkout).
  const stripe = new Stripe(secretKey, { apiVersion: "2026-06-24.dahlia", typescript: true });

  const config = await deployBillingConfig();
  if (!config) {
    throw new Error(
      "deployBillingConfig() returned null — set all STRIPE_LOOKUP_DEPLOY_* env vars to configured, active prices.",
    );
  }

  // A fresh test clock so the created subscription is isolated and time-travelable.
  const clock = await stripe.testHelpers.testClocks.create({
    frozen_time: Math.floor(Date.now() / 1000),
    name: `verify-compute-checkout-${plan}`,
  });
  const customer = await stripe.customers.create({ test_clock: clock.id });

  let subscriptionId: string | undefined;
  try {
    // A test card, set as the default so the first invoice can actually charge.
    const pm = await stripe.paymentMethods.attach("pm_card_visa", { customer: customer.id });
    await stripe.customers.update(customer.id, {
      invoice_settings: { default_payment_method: pm.id },
    });

    // Scenario 6 core — the subscription Checkout would create, made directly so
    // we can inspect its first invoice without the hosted page. Same prices,
    // anchor, billing mode, and proration the checkout page sets.
    const sub = await stripe.subscriptions.create({
      customer: customer.id,
      items: deployCheckoutLineItems(config, plan).map((i) => ({ price: i.price })),
      billing_cycle_anchor_config: { day_of_month: 1 },
      billing_mode: { type: "classic" },
      proration_behavior: "create_prorations",
      expand: ["latest_invoice"],
    });
    subscriptionId = sub.id;

    const anchorDate = sub.billing_cycle_anchor ? new Date(sub.billing_cycle_anchor * 1000) : null;
    const anchorOnFirst = anchorDate?.getUTCDate() === 1;

    const invoice = sub.latest_invoice;
    const firstInvoice = typeof invoice === "string" || !invoice ? null : invoice;
    const amountDueNow = firstInvoice?.amount_due ?? null;
    const amountPaidNow = firstInvoice?.amount_paid ?? null;

    console.info("\n=== Scenario 6 (billing behavior) ===");
    console.info(`plan:                 ${plan}`);
    console.info(`subscription.status:  ${sub.status}`);
    console.info(
      `billing_cycle_anchor: ${anchorDate?.toISOString() ?? "none"}  (on the 1st? ${anchorOnFirst})`,
    );
    console.info(`first invoice amount_due (cents):  ${amountDueNow}`);
    console.info(`first invoice amount_paid (cents): ${amountPaidNow}`);
    console.info(
      amountPaidNow && amountPaidNow > 0
        ? "-> Charges a prorated amount at checkout (R1 'charges at checkout' holds)."
        : "-> First invoice is $0 now; full fee is deferred to the 1st (R1 wording needs revisiting).",
    );
    if (!anchorOnFirst) {
      console.warn(
        "WARNING: anchor did not land on the 1st — billing_cycle_anchor_config may not be honored.",
      );
    }

    // Scenario 1/2/6/7 (hosted page) — build the exact Checkout session the app
    // creates and print its URL. Completing it needs the browser; this confirms
    // the params are accepted and shows the amount Stripe will display.
    const session = await stripe.checkout.sessions.create({
      client_reference_id: "verify-harness",
      billing_address_collection: "auto",
      mode: "subscription",
      line_items: deployCheckoutLineItems(config, plan),
      subscription_data: {
        billing_cycle_anchor_config: { day_of_month: 1 },
        billing_mode: { type: "classic" },
        proration_behavior: "create_prorations",
      },
      customer: customer.id,
      success_url: `${values["base-url"]}/success?session_id={CHECKOUT_SESSION_ID}&intent=deploy&plan=${plan}`,
    });

    console.info("\n=== Scenario 1/2/6/7 (hosted Checkout) ===");
    console.info(`session.mode:         ${session.mode}`);
    console.info(`session.amount_total: ${session.amount_total}`);
    console.info("Open to complete payment and verify the page shows the plan + price:");
    console.info(session.url ?? "(no url)");
  } finally {
    // Clean up: cancel the subscription and delete the clock (removes the customer).
    if (subscriptionId) {
      await stripe.subscriptions.cancel(subscriptionId).catch(() => {});
    }
    await stripe.testHelpers.testClocks.del(clock.id).catch(() => {});
  }
}

main().catch((err) => {
  console.error(err instanceof Error ? err.message : err);
  process.exit(1);
});
