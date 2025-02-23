import { Code } from "@/components/ui/code";
import { insertAuditLogs } from "@/lib/audit";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { PostHogClient } from "@/lib/posthog";
import { currentUser } from "@clerk/nextjs";
import { defaultProSubscriptions } from "@unkey/billing";
import { Empty } from "@unkey/ui";
import { headers } from "next/headers";
import { redirect } from "next/navigation";
import Stripe from "stripe";

type Props = {
  searchParams: {
    session_id: string;
    new_plan: "free" | "pro" | undefined;
  };
};

export default async function StripeSuccess(props: Props) {
  const { session_id, new_plan } = props.searchParams;
  const tenantId = getTenantId();
  const user = await currentUser();
  if (!tenantId || !user) {
    return redirect("/auth/sign-in");
  }

  const ws = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
    with: {
      auditLogBuckets: {
        where: (table, { eq }) => eq(table.name, "unkey_mutations"),
      },
    },
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

  const session = await stripe.checkout.sessions.retrieve(session_id);
  if (!session) {
    return (
      <Empty>
        <Empty.Title>Stripe session not found</Empty.Title>
        <Empty.Description>The Stripe session</Empty.Description>
        <Code>{session_id}</Code>
        <Empty.Description>
          you are trying to access does not exist. Please contact support@unkey.dev.
        </Empty.Description>
      </Empty>
    );
  }
  const customer = await stripe.customers.retrieve(session.customer as string);
  if (!customer) {
    return (
      <Empty>
        <Empty.Title>Stripe customer not found</Empty.Title>
        <Empty.Description>The Stripe customer</Empty.Description>
        <Code>{session.customer as string}</Code>
        <Empty.Description>
          you are trying to access does not exist. Please contact support@unkey.dev.
        </Empty.Description>
      </Empty>
    );
  }

  const isUpgradingPlan = new_plan && new_plan !== ws.plan && new_plan === "pro";
  const h = headers();

  await db.transaction(async (tx) => {
    await tx
      .update(schema.workspaces)
      .set({
        stripeCustomerId: customer.id,
        stripeSubscriptionId: session.subscription as string,
        trialEnds: null,
        ...(isUpgradingPlan
          ? {
              plan: new_plan,
              planChanged: new Date(),
              subscriptions: defaultProSubscriptions(),
              planDowngradeRequest: null,
            }
          : {}),
      })
      .where(eq(schema.workspaces.id, ws.id));

    if (isUpgradingPlan) {
      await insertAuditLogs(tx, ws.auditLogBuckets[0].id, {
        workspaceId: ws.id,
        actor: { type: "user", id: user.id },
        event: "workspace.update",
        description: "Changed plan to 'pro'",
        resources: [
          {
            type: "workspace",
            id: ws.id,
          },
        ],
        context: {
          location: h.get("x-forwarded-for") ?? process.env.VERCEL_REGION ?? "unknown",
          userAgent: h.get("user-agent") ?? undefined,
        },
      });
    }
  });

  if (isUpgradingPlan) {
    PostHogClient.capture({
      distinctId: tenantId,
      event: "plan_changed",
      properties: { plan: new_plan, workspace: ws.id },
    });
  }

  return redirect("/settings/billing");
}
