import { db, eq, schema } from "@unkey/db";
import { getTenantId } from "@/lib/auth";
import Stripe from "stripe";
import { redirect } from "next/navigation";
import { stripeEnv } from "@/lib/env";
import { headers } from "next/headers";

export default async function StripeRedirect() {
  const tenantId = getTenantId();
  if (!tenantId) {
    return redirect("/auth/sign-in");
  }

  const ws = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!ws) {
    return redirect("/onboarding");
  }
  console.log({ ws });
  if (!stripeEnv) {
    return <div>Stripe is not enabled</div>;
  }

  const stripe = new Stripe(stripeEnv.STRIPE_SECRET_KEY!, {
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
    });
    ws.stripeCustomerId = customer.id;
    console.log("updating workspace");

    console.log(
      "query",
      db
        .update(schema.workspaces)
        .set({ stripeCustomerId: customer.id })
        .where(eq(schema.workspaces.id, ws.id))
        .toSQL(),
    );

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
        price: stripeEnv.STRIPE_PRO_PLAN_PRICE_ID,
        quantity: 1,
      },
      {
        // additional keys
        price: stripeEnv.STRIPE_ACTIVE_KEYS_PRICE_ID,
      },
      {
        // additional verifications
        price: stripeEnv.STRIPE_KEY_VERIFICATIONS_PRICE_ID,
      },
    ],
    mode: "subscription",
    success_url: returnUrl,
    cancel_url: returnUrl,
    currency: "USD",
    allow_promotion_codes: true,
    customer: ws.stripeCustomerId,
  });

  return redirect(session.url!);
}
