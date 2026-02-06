import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { Code, Empty } from "@unkey/ui";
import { redirect } from "next/navigation";
import type Stripe from "stripe";

export const dynamic = "force-dynamic";

export default async function StripeRedirect() {
  const { orgId } = await getAuth();

  const ws = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });
  if (!ws) {
    return redirect("/new");
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

  const baseUrl = process.env.VERCEL
    ? process.env.VERCEL_TARGET_ENV === "production"
      ? "https://app.unkey.com"
      : `https://${process.env.VERCEL_URL}`
    : "http://localhost:3000";

  const session = await stripe.checkout.sessions.create({
    client_reference_id: ws.id,
    billing_address_collection: "auto",
    mode: "setup",
    success_url: `${baseUrl}/success?session_id={CHECKOUT_SESSION_ID}`,
    currency: "USD",
    customer_creation: "always",
  });

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

  return redirect(session.url);
}
