import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { getBaseUrl } from "@/lib/utils";
import { Code, Empty } from "@unkey/ui";
import { redirect } from "next/navigation";
import type Stripe from "stripe";

export const dynamic = "force-dynamic";

/**
 * Intents the billing page can attach to a checkout round-trip, so /success
 * knows what the user was actually trying to do. "compute" / "api" reopen
 * that product's plan picker after the card is added; "payment" means the
 * card itself was the goal. Their presence also tells /success to skip the
 * legacy forced API plan modal.
 */
const CHECKOUT_INTENTS = ["compute", "api", "payment"] as const;

export default async function StripeRedirect(props: {
  searchParams: Promise<{ intent?: string }>;
}) {
  const { intent: rawIntent } = await props.searchParams;
  const intent = CHECKOUT_INTENTS.find((known) => known === rawIntent);

  const { orgId, role } = await getAuth();

  if (!orgId) {
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

  // Use the shared `getBaseUrl()` helper so previews resolve to the stable
  // VERCEL_BRANCH_URL rather than a deployment-specific VERCEL_URL.
  const baseUrl = getBaseUrl();

  const session = await stripe.checkout.sessions.create({
    client_reference_id: ws.id,
    billing_address_collection: "auto",
    mode: "setup",
    success_url: `${baseUrl}/success?session_id={CHECKOUT_SESSION_ID}${
      intent ? `&intent=${intent}` : ""
    }`,
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
