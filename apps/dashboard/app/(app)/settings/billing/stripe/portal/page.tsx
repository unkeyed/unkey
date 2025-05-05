import { Code } from "@/components/ui/code";
import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { Empty } from "@unkey/ui";
import { headers } from "next/headers";
import { redirect } from "next/navigation";
import Stripe from "stripe";

export const revalidate = 0;

export default async function StripeRedirect() {
  const { orgId } = await getAuth();
  if (!orgId) {
    return redirect("/auth/sign-in");
  }

  const ws = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
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

  const returnUrl = headers().get("referer") ?? "https://app.unkey.com";

  if (!ws.stripeCustomerId) {
    return (
      <Empty>
        <Empty.Title>No customer found</Empty.Title>
        <Empty.Description>Your workspace</Empty.Description>
        <Code>{ws.id}</Code>
        <Empty.Description>
          is not in Stripe yet. Please contact support@unkey.dev.
        </Empty.Description>
      </Empty>
    );
  }

  const { url } = await stripe.billingPortal.sessions.create({
    customer: ws.stripeCustomerId,
    return_url: returnUrl,
  });
  return redirect(url);
}
