import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { currentUser } from "@clerk/nextjs";
import { headers } from "next/headers";
import { redirect } from "next/navigation";
import Stripe from "stripe";

export default async function StripeRedirect() {
  const tenantId = getTenantId();
  if (!tenantId) {
    return redirect("/auth/sign-in");
  }
  const user = await currentUser();

  const ws = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
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
  if (ws.stripeCustomerId) {
    const session = await stripe.billingPortal.sessions.create({
      customer: ws.stripeCustomerId,
      return_url: headers().get("referer") ?? "https://unkey.dev/app",
    });

    return redirect(session.url);
  }

  // If they don't have a subscription, we send them to the checkout
  // and the checkout will redirect them to the success page, which will add the subscription to the user table
  const baseUrl = process.env.VERCEL_URL ? "https://unkey.dev" : "http://localhost:3000";

  // do not use `new URL(...).searchParams` here, because it will escape the curly braces and stripe will not replace them with the session id
  const successUrl = `${baseUrl}/app/settings/billing/stripe/success?session_id={CHECKOUT_SESSION_ID}`;

  const cancelUrl = headers().get("referer") ?? `${baseUrl}/app`;
  const session = await stripe.checkout.sessions.create({
    client_reference_id: ws.id,
    customer_email: user?.emailAddresses.at(0)?.emailAddress,
    billing_address_collection: "auto",
    mode: "setup",
    success_url: successUrl,
    cancel_url: cancelUrl,
    currency: "USD",
    customer_creation: "always",
  });

  if (!session.url) {
    return <div>Could not create checkout session</div>;
  }

  return redirect(session.url);
}
