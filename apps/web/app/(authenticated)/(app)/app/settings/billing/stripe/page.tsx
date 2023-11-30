import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { currentUser } from "@clerk/nextjs";
import { headers } from "next/headers";
import { redirect } from "next/navigation";
import Stripe from "stripe";
export const runtime = "edge";
export default async function StripeRedirect() {
  const tenantId = getTenantId();
  if (!tenantId) {
    return redirect("/auth/sign-in");
  }
  const user = await currentUser();

  const ws = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!ws) {
    return redirect("/new");
  }
  const e = stripeEnv();
  if (!e) {
    return (
      <EmptyPlaceholder>
        <EmptyPlaceholder.Title>Stripe is not configured</EmptyPlaceholder.Title>
        <EmptyPlaceholder.Description>
          If you are selfhosting Unkey, you need to configure Stripe in your environment variables.
        </EmptyPlaceholder.Description>
      </EmptyPlaceholder>
    );
  }

  const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
    apiVersion: "2022-11-15",
    typescript: true,
  });

  // If they have a subscription already, we display the portal
  if (ws.stripeCustomerId && ws.stripeSubscriptionId) {
    const session = await stripe.billingPortal.sessions.create({
      customer: ws.stripeCustomerId,
    });

    return redirect(session.url);
  }

  // If they don't have a subscription, we send them to the checkout
  // and the checkout will redirect them to the success page, which will add the subscription to the user table
  const baseUrl = process.env.VERCEL_URL
    ? `https://${process.env.VERCEL_URL}`
    : "http://localhost:3000";

  // do not use `new URL(...).searchParams` here, because it will escape the curly braces and stripe will not replace them with the session id
  const successUrl = `${baseUrl}/app/settings/billing/stripe/success?session_id={CHECKOUT_SESSION_ID}`;

  const cancelUrl = headers().get("referer") ?? "https://unkey.dev/app";
  const session = await stripe.checkout.sessions.create({
    client_reference_id: ws.id,
    customer_email: user?.emailAddresses.at(0)?.emailAddress,
    billing_address_collection: "auto",
    line_items: [
      {
        // base
        price: e.STRIPE_PRO_PLAN_PRICE_ID,
        quantity: 1,
      },
      {
        // additional keys
        price: e.STRIPE_ACTIVE_KEYS_PRICE_ID,
      },
      {
        // additional verifications
        price: e.STRIPE_KEY_VERIFICATIONS_PRICE_ID,
      },
    ],
    mode: "subscription",
    success_url: successUrl,
    cancel_url: cancelUrl,
    currency: "USD",
    allow_promotion_codes: true,
    subscription_data: {
      billing_cycle_anchor: nextBillinAnchor(),
    },
  });

  if (!session.url) {
    return <div>Could not create checkout session</div>;
  }

  return redirect(session.url);
}

// Returns midnight of the first day of the next month as unix timestamp (seconds)
function nextBillinAnchor(): number {
  const now = new Date();
  const nextMonth = new Date(now.getFullYear(), now.getMonth() + 1, 1, 0, 0, 0, 0);
  return Math.floor(nextMonth.getTime() / 1000);
}
