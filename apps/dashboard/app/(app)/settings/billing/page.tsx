import { SettingCard } from "@/components/settings-card";
import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { formatNumber } from "@/lib/fmt";
import { Button, Empty, Input } from "@unkey/ui";
import Link from "next/link";
import { redirect } from "next/navigation";
import Stripe from "stripe";
import { Client } from "./client";
import { Shell } from "./components/shell";

export const dynamic = "force-dynamic";
export const revalidate = 0;

export default async function BillingPage() {
  const tenantId = await getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
    with: {
      quota: true,
    },
  });

  if (!workspace) {
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

  const startOfMonth = new Date();
  startOfMonth.setUTCDate(1);
  startOfMonth.setUTCHours(0, 0, 0, 0);

  const year = startOfMonth.getUTCFullYear();
  const month = startOfMonth.getUTCMonth() + 1;

  const [usedVerifications, usedRatelimits] = await Promise.all([
    clickhouse.billing.billableVerifications({
      workspaceId: workspace.id,
      year,
      month,
    }),
    clickhouse.billing.billableRatelimits({
      workspaceId: workspace.id,
      year,
      month,
    }),
  ]);

  const isLegacy = workspace.subscriptions && Object.keys(workspace.subscriptions).length > 0;

  if (isLegacy) {
    return (
      <Shell>
        <div className="w-full">
          <SettingCard
            title="Verifications"
            description="Valid key verifications this month."
            border="top"
          >
            <div className="w-full">
              <Input value={formatNumber(usedVerifications)} />
            </div>
          </SettingCard>
          <SettingCard
            title="Ratelimits"
            description="Valid ratelimits this month."
            border="bottom"
          >
            <div className="w-full">
              <span className="text-xs text-gray-11">
                <Input value={formatNumber(usedRatelimits)} />
              </span>
            </div>
          </SettingCard>
        </div>

        <SettingCard
          title="Legacy plan"
          border="both"
          description={
            <>
              <p>
                You are on the legacy usage-based plan. You can stay on this plan if you want but
                it's likely more expensive than our new{" "}
                <Link href="https://unkey.com/pricing" className="underline" target="_blank">
                  tiered pricing
                </Link>
                .
              </p>
              <p>If you want to switch over, just let us know.</p>
            </>
          }
        >
          <div className="w-full flex justify-end">
            <Button variant="primary" size="lg">
              <Link href="mailto:support@unkey.dev">Contact us</Link>
            </Button>
          </div>
        </SettingCard>
      </Shell>
    );
  }

  const [products, subscription, hasPreviousSubscriptions] = await Promise.all([
    stripe.products
      .list({
        active: true,
        ids: e.STRIPE_PRODUCT_IDS_PRO,
        limit: 100,
        expand: ["data.default_price"],
      })
      .then((res) =>
        res.data
          .map((p) => {
            const price = p.default_price as Stripe.Price;
            return {
              id: p.id,
              name: p.name,
              priceId: price.id,
              dollar: price.unit_amount! / 100,
              quota: {
                requestsPerMonth: Number.parseInt(p.metadata.quota_requests_per_month),
              },
            };
          })
          .sort((a, b) => a.dollar - b.dollar),
      ),
    workspace.stripeSubscriptionId
      ? await stripe.subscriptions.retrieve(workspace.stripeSubscriptionId)
      : undefined,

    workspace.stripeCustomerId
      ? await stripe.subscriptions
          .list({
            customer: workspace.stripeCustomerId,
            status: "canceled",
          })
          .then((res) => res.data.length > 0)
      : false,
  ]);

  return (
    <Client
      hasPreviousSubscriptions={hasPreviousSubscriptions}
      workspace={workspace}
      products={products}
      usage={{
        current: usedVerifications + usedRatelimits,
        max: workspace.quota?.requestsPerMonth ?? 150_000,
      }}
      subscription={
        subscription
          ? {
              id: subscription.id,
              status: subscription.status,
              trialUntil: subscription.trial_end ? subscription.trial_end * 1000 : undefined,
              cancelAt: subscription.cancel_at ? subscription.cancel_at * 1000 : undefined,
            }
          : undefined
      }
      currentProductId={subscription?.items.data.at(0)?.plan.product?.toString() ?? undefined}
    />
  );
}
