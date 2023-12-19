import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Code } from "@/components/ui/code";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { currentUser } from "@clerk/nextjs";
import { redirect } from "next/navigation";
import Stripe from "stripe";

type Props = {
  searchParams: {
    session_id: string;
  };
};

export default async function StripeSuccess(props: Props) {
  const tenantId = getTenantId();
  if (!tenantId) {
    return redirect("/auth/sign-in");
  }
  const _user = await currentUser();

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

  const session = await stripe.checkout.sessions.retrieve(props.searchParams.session_id);
  if (!session) {
    return (
      <EmptyPlaceholder>
        <EmptyPlaceholder.Title>Stripe session not found</EmptyPlaceholder.Title>
        <EmptyPlaceholder.Description>
          The Stripe session <Code>{props.searchParams.session_id}</Code> you are trying to access
          does not exist. Please contact support@unkey.dev.
        </EmptyPlaceholder.Description>
      </EmptyPlaceholder>
    );
  }
  const customer = await stripe.customers.retrieve(session.customer as string);
  if (!customer) {
    return (
      <EmptyPlaceholder>
        <EmptyPlaceholder.Title>Stripe session not found</EmptyPlaceholder.Title>
        <EmptyPlaceholder.Description>
          The Stripe customer <Code>{session.customer as string}</Code> you are trying to access
          does not exist. Please contact support@unkey.dev.
        </EmptyPlaceholder.Description>
      </EmptyPlaceholder>
    );
  }

  await db
    .update(schema.workspaces)
    .set({
      stripeCustomerId: customer.id,
      stripeSubscriptionId: session.subscription as string,
      trialEnds: null,
    })
    .where(eq(schema.workspaces.id, ws.id));

  return redirect("/app/settings/billing");
}
