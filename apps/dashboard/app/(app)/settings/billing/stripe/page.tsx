import { getTenantId } from "@/lib/auth";
import { auth } from "@/lib/auth/server";
import { db } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { Empty } from "@unkey/ui";
import { headers } from "next/headers";
import { redirect } from "next/navigation";
import Stripe from "stripe";

type Props = {
  searchParams: {
    new_plan: "free" | "pro" | undefined;
  };
};

export default async function StripeRedirect(props: Props) {
  const { new_plan } = props.searchParams;
  const user = await auth.getCurrentUser();
  const tenantId = await getTenantId();

  const ws = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
  });
  if (!ws) {
    return redirect("/new");
  }
  const e = stripeEnv();
  if (!e) {
    return (
      <Empty>
        <Empty.Title>Stripe is not configured</Empty.Title>
        <Empty.Description>
          If you are selfhosting Unkey, you need to configure Stripe in your environment variables.
        </Empty.Description>
      </Empty>
    );
  }

  const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
    apiVersion: "2023-10-16",
    typescript: true,
  });

  // If they have a subscription already, we display the portal
  if (ws.stripeCustomerId) {
    const session = await stripe.billingPortal.sessions.create({
      customer: ws.stripeCustomerId,
      return_url: headers().get("referer") ?? "https://app.unkey.com",
    });

    return redirect(session.url);
  }

  // If they don't have a subscription, we send them to the checkout
  // and the checkout will redirect them to the success page, which will add the subscription to the user table
  const baseUrl = process.env.VERCEL_URL ? "https://app.unkey.com" : "http://localhost:3000";

  // do not use `new URL(...).searchParams` here, because it will escape the curly braces and stripe will not replace them with the session id
  let successUrl = `${baseUrl}/settings/billing/stripe/success?session_id={CHECKOUT_SESSION_ID}`;

  // if they're coming from the change plan flow, pass along the new plan param
  if (new_plan && new_plan !== ws.plan) {
    successUrl += `&new_plan=${new_plan}`;
  }

  const cancelUrl = headers().get("referer") ?? baseUrl;
  const session = await stripe.checkout.sessions.create({
    client_reference_id: ws.id,
    customer_email: user?.email,
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
