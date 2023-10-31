import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
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
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!ws) {
    return redirect("/onboarding");
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

  // else we upsert them and then display checkout

  if (!ws.stripeCustomerId) {
    const customer = await stripe.customers.create({
      name: ws.name,
      email: user?.emailAddresses.at(0)?.emailAddress,
    });
    ws.stripeCustomerId = customer.id;

    await db
      .update(schema.workspaces)
      .set({ stripeCustomerId: customer.id })
      .where(eq(schema.workspaces.id, ws.id));
  }
  const returnUrl = headers().get("referer") ?? "https://unkey.dev/app";

  const session = await stripe.checkout.sessions.create({
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
    success_url: returnUrl,
    cancel_url: returnUrl,
    currency: "USD",
    allow_promotion_codes: true,
    customer: ws.stripeCustomerId,
  });

  if (!session.url) {
    return <div>Could not create checkout session</div>;
  }

  return redirect(session.url);
}
