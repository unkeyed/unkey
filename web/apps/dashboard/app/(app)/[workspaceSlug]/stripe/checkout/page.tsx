import { getAuth } from "@/lib/auth";
import { parseReturnPath } from "@/lib/billing/upgrade-result";
import { db } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { formatDollars } from "@/lib/fmt";
import { DEPLOY_CHECKOUT_ORIGINS, routes } from "@/lib/navigation/routes";
import { getStripeClient } from "@/lib/stripe";
import { subscriptionIdsByProduct } from "@/lib/stripe/billingSubscriptions";
import {
  ComputeSignupCheckoutError,
  type DeployCheckoutCredit,
  deployCardSetupSuccessUrl,
  deployCheckoutCustomerId,
  deployCheckoutSubmitMessage,
  prepareDeployCheckoutCredit,
} from "@/lib/stripe/computeSignupCheckout";
import {
  COMPUTE_SIGNUP_CREDIT_CENTS,
  ComputeSignupCreditRetryError,
  signupWorkosUserId,
} from "@/lib/stripe/computeSignupCredit";
import { createSubscriptionCheckout } from "@/lib/stripe/createSubscriptionCheckout";
import { deployBillingConfig, deployCheckoutLineItems } from "@/lib/stripe/deployBilling";
import { DEPLOY_PLANS } from "@/lib/stripe/deployPlan";
import { hostedInvoiceUrl, isDeadSubscription } from "@/lib/stripe/subscriptionUtils";
import { getBaseUrl } from "@/lib/utils";
import {
  Code,
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  PageBody,
  PageContainer,
} from "@unkey/ui";
import type { Route } from "next";
import { redirect } from "next/navigation";
import Stripe from "stripe";

export const dynamic = "force-dynamic";

const CHECKOUT_INTENTS = ["payment", "deploy"] as const;

export default async function StripeRedirect(props: {
  searchParams: Promise<{
    intent?: string;
    plan?: string;
    from?: string;
    returnTo?: string;
    setup_session_id?: string;
  }>;
}) {
  const {
    intent: rawIntent,
    plan: rawPlan,
    from: rawFrom,
    returnTo: rawReturnTo,
    setup_session_id: setupSessionId,
  } = await props.searchParams;
  const intent = CHECKOUT_INTENTS.find((known) => known === rawIntent);
  const plan = DEPLOY_PLANS.find((known) => known === rawPlan);
  const from = DEPLOY_CHECKOUT_ORIGINS.find((known) => known === rawFrom);

  const { orgId, role, userId } = await getAuth();

  if (!orgId) {
    // route-guard-ignore: pre-existing unauthenticated redirect, left untouched
    return redirect("/sign-in");
  }

  // Mirror the client-side admin gate. The Add-payment-method button is
  // hidden for non-admins, but this page is reachable directly via URL.
  if (role !== "admin") {
    return (
      <PageContainer>
        <PageBody>
          <EmptyState>
            <EmptyStateHeader>
              <EmptyStateTitle>Admin access required</EmptyStateTitle>
              <EmptyStateDescription>
                Only workspace admins can manage billing. Ask an admin to make changes.
              </EmptyStateDescription>
            </EmptyStateHeader>
          </EmptyState>
        </PageBody>
      </PageContainer>
    );
  }

  const ws = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    columns: { id: true, slug: true },
    with: {
      billing: {
        columns: { stripeCustomerId: true },
      },
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
      <PageContainer>
        <PageBody>
          <EmptyState>
            <EmptyStateHeader>
              <EmptyStateTitle>Stripe is not configured</EmptyStateTitle>
              <EmptyStateDescription>
                If you are selfhosting Unkey, you need to configure Stripe in your environment
                variables.
              </EmptyStateDescription>
            </EmptyStateHeader>
          </EmptyState>
        </PageBody>
      </PageContainer>
    );
  }

  // Use the shared `getBaseUrl()` helper so previews resolve to the stable
  // VERCEL_BRANCH_URL rather than a deployment-specific VERCEL_URL.
  const baseUrl = getBaseUrl();
  const existingCustomerId = ws.billing?.stripeCustomerId ?? undefined;
  const returnPath = intent === "deploy" ? parseReturnPath(rawReturnTo, ws.slug) : null;

  const successUrl = `${baseUrl}/success?session_id={CHECKOUT_SESSION_ID}${
    intent ? `&intent=${intent}` : ""
  }${intent === "deploy" && plan ? `&plan=${plan}` : ""}${
    intent === "deploy" && from ? `&from=${from}` : ""
  }${returnPath ? `&returnTo=${encodeURIComponent(returnPath)}` : ""}`;

  // Dev/test only: Checkout cannot create customers under a Stripe test
  // clock, so when STRIPE_DEV_TEST_CLOCK is set we create a clocked customer
  // up front and hand it to the session. That makes every workspace set up
  // through the UI time-travelable: advance the clock and its invoices
  // finalize for real (PDF included). One clock per customer, since a clock
  // carries at most a handful of customers and advances them together. Reuse
  // an existing workspace customer first so Checkout can show its saved cards
  // and API and Compute remain under the same Stripe customer. A return from
  // card setup already has that customer on the setup session. Minting another
  // clocked customer on that load would fail the session customer check.
  let devClockedCustomerId: string | undefined;
  if (!existingCustomerId && !setupSessionId && stripeEnv()?.STRIPE_DEV_TEST_CLOCK === "true") {
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
  const checkoutCustomerId = deployCheckoutCustomerId({
    existingCustomerId,
    devClockedCustomerId,
    setupSessionId,
  });

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
    const payer = signupWorkosUserId(userId);
    let creditStep: DeployCheckoutCredit;
    try {
      creditStep = await prepareDeployCheckoutCredit(stripe, {
        workspaceId: ws.id,
        workosUserId: userId,
        customerId: checkoutCustomerId,
        setupSessionId,
        nowMs: Date.now(),
      });
    } catch (error) {
      if (error instanceof ComputeSignupCreditRetryError) {
        return checkoutMessage(
          "Applying your credit",
          "Your $5 Compute credit is still being applied. Refresh this page to continue checkout.",
        );
      }
      if (error instanceof ComputeSignupCheckoutError) {
        return checkoutMessage("Checkout unavailable", error.message);
      }
      throw error;
    }

    if (creditStep.step === "collect_card" && payer) {
      session = await stripe.checkout.sessions.create({
        client_reference_id: ws.id,
        billing_address_collection: "auto",
        mode: "setup",
        payment_method_types: ["card"],
        success_url: deployCardSetupSuccessUrl({
          baseUrl,
          workspaceSlug: ws.slug,
          plan,
          from,
          returnTo: returnPath ?? undefined,
        }),
        currency: "USD",
        setup_intent_data: { metadata: { workos_user_id: payer } },
        ...(checkoutCustomerId
          ? { customer: checkoutCustomerId }
          : { customer_creation: "always" as const }),
      });
    } else {
      const subscribeCustomerId =
        creditStep.step === "subscribe" ? creditStep.customerId : checkoutCustomerId;
      let planFee: string | undefined;
      try {
        const price = await stripe.prices.retrieve(deployConfig.planFeePriceIds[plan]);
        if (price.unit_amount != null) {
          planFee = formatDollars(price.unit_amount);
        }
      } catch {
        // The plan fee is Stripe's number. Checkout still proceeds.
      }
      const submitMessage = deployCheckoutSubmitMessage({
        announceSignupCredit: creditStep.step === "subscribe" && creditStep.announceSignupCredit,
        signupCredit: formatDollars(COMPUTE_SIGNUP_CREDIT_CENTS),
        planFee,
      });

      const destination = await createSubscriptionCheckout(stripe, {
        workspaceId: ws.id,
        product: "compute",
        ...(subscribeCustomerId ? { customerId: subscribeCustomerId } : {}),
        lineItems: deployCheckoutLineItems(deployConfig, plan),
        successUrl,
        ...(payer ? { workosUserId: payer } : {}),
        ...(submitMessage ? { customText: { submit: { message: submitMessage } } } : {}),
        ...(devClockedCustomerId
          ? {}
          : {
              idempotencyKey: `deploy-checkout:${ws.id}:${plan}:${from ?? ""}:${subscribeCustomerId ?? ""}`,
            }),
      });
      if (destination.kind === "success") {
        return redirect(destination.url as Route);
      }
      session = destination.session;
    }
  } else {
    session = await stripe.checkout.sessions.create({
      client_reference_id: ws.id,
      billing_address_collection: "auto",
      mode: "setup",
      payment_method_types: ["card"],
      success_url: successUrl,
      currency: "USD",
      setup_intent_data: { metadata: { workos_user_id: signupWorkosUserId(userId) ?? userId } },
      ...(checkoutCustomerId
        ? { customer: checkoutCustomerId }
        : { customer_creation: "always" as const }),
    });
  }

  if (!session.url) {
    return (
      <PageContainer>
        <PageBody>
          <EmptyState>
            <EmptyStateHeader>
              <EmptyStateTitle>Empty Session</EmptyStateTitle>
              <EmptyStateDescription>The Stripe session</EmptyStateDescription>
            </EmptyStateHeader>
            <Code>{session.id}</Code>
            <EmptyStateDescription>
              you are trying to access does not exist. Please contact support@unkey.com.
            </EmptyStateDescription>
          </EmptyState>
        </PageBody>
      </PageContainer>
    );
  }

  return redirect(session.url as Route);
}

function checkoutMessage(title: string, description: string) {
  return (
    <PageContainer>
      <PageBody>
        <EmptyState>
          <EmptyStateHeader>
            <EmptyStateTitle>{title}</EmptyStateTitle>
            <EmptyStateDescription>{description}</EmptyStateDescription>
          </EmptyStateHeader>
        </EmptyState>
      </PageBody>
    </PageContainer>
  );
}
