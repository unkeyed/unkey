import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { formatDollars } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import { getStripeClient } from "@/lib/stripe";
import { deployBillingConfig, deployCheckoutLineItems } from "@/lib/stripe/deployBilling";
import { DEPLOY_PLANS } from "@/lib/stripe/deployPlan";
import { isDeadSubscription } from "@/lib/stripe/subscriptionUtils";
import { getBaseUrl } from "@/lib/utils";
import { Code, Empty } from "@unkey/ui";
import type { Route } from "next";
import { redirect } from "next/navigation";
import Stripe from "stripe";

export const dynamic = "force-dynamic";

/**
 * Intents the billing page can attach to a checkout round-trip, so /success
 * knows what the user was actually trying to do. "compute" / "api" reopen
 * that product's plan picker after the card is added; "payment" means the
 * card itself was the goal. Their presence also tells /success to skip the
 * legacy forced API plan modal. "deploy" comes from the compute-plan gate and
 * carries `plan`/`from` so /success can return the user to the projects page
 * and subscribe there.
 */
const CHECKOUT_INTENTS = ["compute", "api", "payment", "deploy"] as const;
const DEPLOY_ORIGINS = ["create", "banner", "billing"] as const;

export default async function StripeRedirect(props: {
  searchParams: Promise<{ intent?: string; plan?: string; from?: string }>;
}) {
  const { intent: rawIntent, plan: rawPlan, from: rawFrom } = await props.searchParams;
  const intent = CHECKOUT_INTENTS.find((known) => known === rawIntent);
  const plan = DEPLOY_PLANS.find((known) => known === rawPlan);
  const from = DEPLOY_ORIGINS.find((known) => known === rawFrom);

  const { orgId, role } = await getAuth();

  if (!orgId) {
    // route-guard-ignore: pre-existing unauthenticated redirect, left untouched
    return redirect("/sign-in");
  }

  // Mirror the client-side admin gate. The Add-payment-method button is
  // hidden for non-admins, but this page is reachable directly via URL.
  if (role !== "admin") {
    return (
      <Empty>
        <Empty.Title>Admin access required</Empty.Title>
        <Empty.Description>
          Only workspace admins can manage billing. Ask an admin to make changes.
        </Empty.Description>
      </Empty>
    );
  }

  const ws = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });
  if (!ws) {
    return redirect(routes.workspaces.create());
  }

  let stripe: Stripe;
  try {
    stripe = getStripeClient();
  } catch (_error) {
    return (
      <Empty>
        <Empty.Title>Stripe is not configured</Empty.Title>
        <Empty.Description>
          If you are selfhosting Unkey, you need to configure Stripe in your environment variables.
        </Empty.Description>
      </Empty>
    );
  }

  // Use the shared `getBaseUrl()` helper so previews resolve to the stable
  // VERCEL_BRANCH_URL rather than a deployment-specific VERCEL_URL.
  const baseUrl = getBaseUrl();

  const successUrl = `${baseUrl}/success?session_id={CHECKOUT_SESSION_ID}${
    intent ? `&intent=${intent}` : ""
  }${intent === "deploy" && plan ? `&plan=${plan}` : ""}${
    intent === "deploy" && from ? `&from=${from}` : ""
  }`;

  // Dev/test only: Checkout cannot create customers under a Stripe test
  // clock, so when STRIPE_DEV_TEST_CLOCK is set we create a clocked customer
  // up front and hand it to the session. That makes every workspace set up
  // through the UI time-travelable: advance the clock and its invoices
  // finalize for real (PDF included). One clock per customer, since a clock
  // carries at most a handful of customers and advances them together.
  let devClockedCustomerId: string | undefined;
  if (stripeEnv()?.STRIPE_DEV_TEST_CLOCK === "true") {
    const clock = await stripe.testHelpers.testClocks.create({
      frozen_time: Math.floor(Date.now() / 1000),
      name: ws.slug,
    });
    const customer = await stripe.customers.create({
      test_clock: clock.id,
      metadata: { workspace_id: ws.id },
    });
    devClockedCustomerId = customer.id;
  }

  // For the Compute-plan gate's no-card path, create the subscription in
  // Checkout itself (mode: "subscription") so Stripe shows the plan name and
  // monthly price and charges at checkout. Every other intent — and a
  // workspace that already has a LIVE subscription, to avoid creating a second
  // one — falls through to the card-vault setup session below. A dead recorded
  // subscription (cancelDeploy cancels a Compute-only subscription outright,
  // and the deleted-webhook that clears the column may lag) counts as absent,
  // or a mid-month cancel could never resubscribe. deployBillingConfig returns
  // null when Compute billing is unconfigured, which also falls back.
  let hasLiveSubscription = false;
  if (intent === "deploy" && plan && ws.stripeSubscriptionId) {
    // A recorded subscription that no longer exists on Stripe is the same
    // "dead recorded subscription counts as absent" case, not a 500; mirrors
    // linkDeploySubscription. Anything else propagates — a transient failure
    // must not silently downgrade a live subscription to "absent".
    const recorded = await stripe.subscriptions
      .retrieve(ws.stripeSubscriptionId)
      .catch((err: unknown) => {
        if (err instanceof Stripe.errors.StripeError && err.code === "resource_missing") {
          return null;
        }
        throw err;
      });
    hasLiveSubscription = recorded !== null && !isDeadSubscription(recorded);
  }
  const deployConfig =
    intent === "deploy" && plan && !hasLiveSubscription ? await deployBillingConfig() : null;

  let session: Stripe.Checkout.Session;
  if (deployConfig && plan) {
    // Resolve the selected plan's fee so the credits message names the right
    // amount (credits equal the fee). Omit the message rather than fail the
    // session if the price can't be resolved.
    let submitMessage: string | undefined;
    try {
      const price = await stripe.prices.retrieve(deployConfig.planFeePriceIds[plan]);
      if (price.unit_amount != null) {
        const amount = formatDollars(price.unit_amount);
        // Credits equal the plan fee actually charged on each invoice
        // (netDeployFee sums the fee lines), so they are prorated at checkout
        // and full each month. Word the message as "credits match the charge"
        // rather than a fixed number, since Stripe itself shows the prorated
        // amount due today and we do not recompute its proration here.
        submitMessage = `Your plan fee is matched by usage credits: ${amount} each month, and a prorated first charge is matched by the same amount in credits.`;
      }
    } catch {
      // Non-fatal: proceed without the credits message.
    }

    // billing_cycle_anchor_config on Checkout requires API version
    // 2026-06-24.dahlia or later, which is the version getStripeClient pins
    // (stripe-node types the constructor apiVersion as exactly the bundled
    // version, so the whole client is on 2026-06-24.dahlia).
    const sessionParams: Stripe.Checkout.SessionCreateParams = {
      client_reference_id: ws.id,
      billing_address_collection: "auto",
      mode: "subscription",
      line_items: deployCheckoutLineItems(deployConfig, plan),
      subscription_data: {
        // Match subscribeDeploy's shape so a Checkout-created Compute
        // subscription is the same as one it creates: day-1 anchor and classic
        // billing mode. subscribeDeploy pins proration_behavior "always_invoice",
        // which Checkout does not accept; "create_prorations" is the closest
        // Checkout-valid behavior and still collects the prorated partial period
        // on the first invoice at checkout.
        billing_cycle_anchor_config: { day_of_month: 1 },
        billing_mode: { type: "classic" },
        proration_behavior: "create_prorations",
      },
      ...(submitMessage ? { custom_text: { submit: { message: submitMessage } } } : {}),
      // Subscription mode always creates a customer (so customer_creation is
      // invalid here) and infers currency from the line-item prices.
      ...(devClockedCustomerId ? { customer: devClockedCustomerId } : {}),
      success_url: successUrl,
    };

    if (devClockedCustomerId) {
      // No idempotency key under the dev test clock, which mints a fresh
      // customer per request (params differ every time and the key would
      // conflict).
      session = await stripe.checkout.sessions.create(sessionParams);
    } else {
      // Idempotency key so a retry within Stripe's window returns the SAME
      // session instead of creating a second live, charged subscription — the
      // race where the user pays, abandons before the link is written, then
      // re-opens the gate (stripeSubscriptionId still null). Keyed by workspace
      // + plan + origin, since success_url varies by `from` and a differing
      // param under the same key would trip an idempotency mismatch.
      //
      // Stripe's idempotency layer replays the CREATION-TIME response, so a
      // replayed session always reads status "open" with a working-looking url
      // even if it has since been paid or expired — redirecting to it then
      // shows Stripe's "this checkout session has timed out" dead end. So
      // re-retrieve for the live status and branch:
      //  - open:     the normal redirect below.
      //  - complete with a LIVE subscription: it was PAID (the
      //    abandon-before-link race) — hand off to /success for this session,
      //    which links the subscription instead of charging a second time.
      //  - expired, or complete with a dead subscription (a finished
      //    subscribe→cancel cycle): nothing to resume; chain a new
      //    deterministic key off the stale session id and mint a fresh session
      //    (a retry of THIS request replays the same fresh session rather than
      //    double-creating).
      let idempotencyKey = `deploy-checkout:${ws.id}:${plan}:${from ?? ""}`;
      session = await stripe.checkout.sessions.create(sessionParams, { idempotencyKey });
      for (let attempt = 0; attempt < 3; attempt++) {
        const live = await stripe.checkout.sessions.retrieve(session.id);
        if (live.status === "open") {
          break;
        }
        if (live.status === "complete") {
          const paidSubId =
            typeof live.subscription === "string" ? live.subscription : live.subscription?.id;
          const paidSub = paidSubId ? await stripe.subscriptions.retrieve(paidSubId) : null;
          if (paidSub && !isDeadSubscription(paidSub)) {
            return redirect(successUrl.replace("{CHECKOUT_SESSION_ID}", live.id) as Route);
          }
        }
        idempotencyKey = `${idempotencyKey}:${session.id}`;
        session = await stripe.checkout.sessions.create(sessionParams, { idempotencyKey });
      }
    }
  } else {
    session = await stripe.checkout.sessions.create({
      client_reference_id: ws.id,
      billing_address_collection: "auto",
      mode: "setup",
      success_url: successUrl,
      currency: "USD",
      ...(devClockedCustomerId
        ? { customer: devClockedCustomerId }
        : { customer_creation: "always" as const }),
    });
  }

  if (!session.url) {
    return (
      <Empty>
        <Empty.Title>Empty Session</Empty.Title>
        <Empty.Description>The Stripe session</Empty.Description>
        <Code>{session.id}</Code>
        <Empty.Description>
          you are trying to access does not exist. Please contact support@unkey.com.
        </Empty.Description>
      </Empty>
    );
  }

  return redirect(session.url as Route);
}
