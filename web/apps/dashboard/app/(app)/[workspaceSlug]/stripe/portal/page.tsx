import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { routes } from "@/lib/navigation/routes";
import { getStripeClient } from "@/lib/stripe";
import { getBaseUrl } from "@/lib/utils";
import { Code, Empty } from "@unkey/ui";
import type { Route } from "next";
import { redirect } from "next/navigation";
import type Stripe from "stripe";

export const dynamic = "force-dynamic";

export default async function StripeRedirect() {
  const { orgId, role } = await getAuth();

  if (!orgId) {
    // route-guard-ignore: pre-existing unauthenticated redirect, left untouched
    return redirect("/sign-in");
  }

  // Mirror the client-side admin gate. Even though the dashboard now hides
  // the portal button for non-admins, this page is reachable directly via
  // URL — refuse to mint a Stripe billing-portal session for non-admins.
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
    return redirect(routes.workspaces.create());
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
  // VERCEL_BRANCH_URL (e.g. dashboard-git-<branch>-unkey.vercel.app) rather
  // than a deployment-specific VERCEL_URL that changes on every push and
  // breaks links the user already has open.
  const baseUrl = getBaseUrl();

  if (!ws.stripeCustomerId) {
    return (
      <Empty>
        <Empty.Title>No customer found</Empty.Title>
        <Empty.Description>Your workspace</Empty.Description>
        <Code>{ws.id}</Code>
        <Empty.Description>
          is not in Stripe yet. Please contact support@unkey.com.
        </Empty.Description>
      </Empty>
    );
  }

  // Return the user back to the workspace's billing page. /success is the
  // post-checkout landing page and isn't the right destination for the
  // billing-portal round-trip — it has no `session_id` here, so it shows an
  // empty state, and (depending on auth state after the Stripe round-trip)
  // can bounce the user to sign-in.
  const { url } = await stripe.billingPortal.sessions.create({
    customer: ws.stripeCustomerId,
    return_url: `${baseUrl}${routes.settings.billing({ workspaceSlug: ws.slug })}`,
  });
  return redirect(url as Route);
}
