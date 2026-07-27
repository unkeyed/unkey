import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { formatDollars } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import { getStripeClient } from "@/lib/stripe";
import { subscriptionIdsByProduct } from "@/lib/stripe/billingSubscriptions";
import { createSubscriptionCheckout } from "@/lib/stripe/createSubscriptionCheckout";
import { deployBillingConfig, deployCheckoutLineItems } from "@/lib/stripe/deployBilling";
import { DEPLOY_PLANS } from "@/lib/stripe/deployPlan";
import { hostedInvoiceUrl, isDeadSubscription } from "@/lib/stripe/subscriptionUtils";
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
    columns: { id: true, slug: true },
    with: {
      billingSubscriptions: {
        columns: { product: true, stripeSubscriptionId: true },
      },
    },
  });
  if (!ws) {
    return redirect(routes.workspaces.create());
  }

  const stripeDeploySubscriptionId = subscriptionIdsByProduct(
    ws.billingSubscriptions ?? [],
  ).stripeDeploySubscriptionId;

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
  const existingCustomerId = ws.billing?.stripeCustomerId ?? undefined;

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
  // carries at most a handful of customers and advances them together. Reuse
  // an existing workspace customer first so Checkout can show its saved cards
  // and API and Compute remain under the same Stripe customer.
  let devClockedCustomerId: string | undefined;
  if (!existingCustomerId && stripeEnv()?.STRIPE_DEV_TEST_CLOCK === "true") {
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
  // Keep API and Compute subscriptions on the workspace's existing Stripe
  // customer. This path is also used to replace a bad vaulted card; creating a
  // new customer there would strand the existing API subscription on the old
  // customer and make the portal appear to lose one of the products.
  const checkoutCustomerId = existingCustomerId ?? devClockedCustomerId;

  // Create a selected Compute plan in subscription-mode Checkout even when the
  // customer already has a saved card, so Stripe shows the plan and can handle
  // CVC, replacement cards, or 3DS. Every other intent — and a
  // workspace that already has a LIVE Deploy subscription, to avoid creating a
  // second one — falls through to the card-vault setup session below. A dead
  // recorded subscription (cancelDeploy cancels the Compute subscription
  // outright, and the deleted-webhook that clears the column may lag) counts as
  // absent, or a mid-month cancel could never resubscribe. deployBillingConfig
  // returns null when Compute billing is unconfigured, which also falls back.
  let hasLiveSubscription = false;
  if (intent === "deploy" && plan && stripeDeploySubscriptionId) {
    // A recorded subscription that no longer exists on Stripe is the same
    // "dead recorded subscription counts as absent" case, not a 500; mirrors
    // linkDeploySubscription. Anything else propagates — a transient failure
    // must not silently downgrade a live subscription to "absent".
    let recorded = await stripe.subscriptions
      .retrieve(stripeDeploySubscriptionId, { expand: ["latest_invoice"] })
      .catch((err: unknown) => {
        if (err instanceof Stripe.errors.StripeError && err.code === "resource_missing") {
          return null;
        }
        throw err;
      });
    if (recorded?.status === "incomplete") {
      const paymentUrl = hostedInvoiceUrl(recorded);
      if (paymentUrl) {
        return redirect(paymentUrl as Route);
      }
      recorded = await stripe.subscriptions.cancel(recorded.id);
    }
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

    const destination = await createSubscriptionCheckout(stripe, {
      workspaceId: ws.id,
      product: "compute",
      customerId: checkoutCustomerId,
      lineItems: deployCheckoutLineItems(deployConfig, plan),
      successUrl,
      ...(submitMessage ? { customText: { submit: { message: submitMessage } } } : {}),
      ...(devClockedCustomerId
        ? {}
        : {
            idempotencyKey: `deploy-checkout:${ws.id}:${plan}:${from ?? ""}:${checkoutCustomerId ?? "new"}`,
          }),
    });
    if (destination.kind === "success") {
      return redirect(destination.url as Route);
    }
    session = destination.session;
  } else {
    session = await stripe.checkout.sessions.create({
      client_reference_id: ws.id,
      billing_address_collection: "auto",
      mode: "setup",
      success_url: successUrl,
      currency: "USD",
      ...(checkoutCustomerId
        ? { customer: checkoutCustomerId }
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
