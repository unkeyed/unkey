import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { Navigation } from "@/components/navigation/navigation";
import { PageContent } from "@/components/page-content";
import { SettingCard } from "@/components/settings-card";
import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { Gear } from "@unkey/icons";
import { Button, Empty, Input } from "@unkey/ui";
import ms from "ms";
import Link from "next/link";
import { redirect } from "next/navigation";
import type { PropsWithChildren } from "react";
import Stripe from "stripe";
import { navigation } from "../constants";

export const revalidate = 0;

export default async function BillingPage() {
  const tenantId = getTenantId();

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

  const sub = workspace.stripeSubscriptionId
    ? await stripe.subscriptions.retrieve(workspace.stripeSubscriptionId)
    : null;

  const isLegacy = workspace.subscriptions && Object.keys(workspace.subscriptions).length > 0;
  // isNew means a workspace has not had a trial or paid plan yet and we should create one
  const isNew = !workspace.stripeCustomerId;

  const isCancelled = !!sub?.cancel_at;

  let trialNotice: string | null = null;
  if (sub?.trial_end) {
    const remainingMs = sub.trial_end * 1000 - Date.now();
    if (remainingMs > 0) {
      trialNotice = `Your Trial ends in ${ms(remainingMs)}`;
    }
  }

  if (isLegacy) {
    return (
      <Shell>
        <LegacyUsage workspaceId={workspace.id} />

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

  if (isNew) {
    return (
      <Shell>
        <Empty className="border border-gray-4 rounded-xl">
          <Empty.Title>You are on the Free tier.</Empty.Title>
          <Empty.Description>
            <p>The Free tier includes a fixed amount of free usage.</p>
            <p>
              To unlock additional usage and add team members, upgrade to Pro.{" "}
              <Link
                href="https://unkey.com/pricing"
                target="_blank"
                className="text-info-11 underline"
              >
                See Pricing
              </Link>
            </p>
          </Empty.Description>
        </Empty>
        <Usage
          workspaceId={workspace.id}
          maxRequestsQuota={workspace.quota?.requestsPerMonth ?? 250000}
        />
        <SettingCard
          className="border-success-7 bg-success-2"
          title="Start trial"
          border="both"
          description="Add a payment method to start a 14 day trial for the Pro tier. You can cancel at any time. "
        >
          <div className="w-full flex justify-end">
            <Button variant="primary" size="lg">
              <Link href="/settings/billing/stripe?action=payment_intent">Start Trial</Link>
            </Button>
          </div>
        </SettingCard>
      </Shell>
    );
  }

  return (
    <Shell>
      {sub?.cancel_at && sub.cancel_at > Date.now() / 1000 ? (
        <SettingCard
          title="Cancelled"
          description={`Your plan is scheduled to be cancelled on ${new Date(
            sub.cancel_at * 1000,
          ).toDateString()}`}
          border="both"
          className="border-warning-7 bg-warning-2"
        >
          <div className="w-full flex justify-end">
            <Button size="lg" variant="primary" color="warning">
              <Link href="/settings/billing/stripe?action=portal">Resubscribe</Link>
            </Button>
          </div>
        </SettingCard>
      ) : null}

      <Usage
        workspaceId={workspace.id}
        maxRequestsQuota={workspace.quota?.requestsPerMonth ?? 250000}
      />
      <div className="w-full divide-y divide-gray-4">
        <SettingCard
          title="Plan"
          border="top"
          description={
            <>
              <p>
                You are on the <strong className="capitalize">{workspace.tier}</strong> plan.
              </p>
              <p>{trialNotice}</p>
            </>
          }
        >
          <div className="w-full flex justify-end">
            <Button size="lg" variant="outline">
              <Link href="/settings/billing/stripe?action=subscription_update">Change Plan</Link>
            </Button>
          </div>
        </SettingCard>
        <SettingCard
          title="Billing Portal"
          border="bottom"
          description="Manage everything in Stripe's portal."
        >
          <div className="w-full flex justify-end">
            <Button className="w-fit rounded-lg" variant="outline" size="lg">
              <Link href="/settings/billing/stripe?action=portal">Open Portal</Link>
            </Button>
          </div>
        </SettingCard>
      </div>

      {sub && !isCancelled ? (
        <SettingCard
          title="Cancel Subscription"
          description="Cancel your subscription at the end of the current month."
          border="both"
        >
          <div className="w-full flex justify-end">
            <Button className="w-fit rounded-lg" variant="outline" color="danger" size="lg">
              <Link href="/settings/billing/stripe?action=subscription_cancel">
                Cancel Subscription
              </Link>
            </Button>
          </div>
        </SettingCard>
      ) : null}
    </Shell>
  );
}

const Shell: React.FC<PropsWithChildren> = ({ children }) => {
  return (
    <div>
      <Navigation href="/settings/billing" name="Settings" icon={<Gear />} />
      <PageContent>
        <SubMenu navigation={navigation} segment="billing" />
        <div className="py-3 w-full flex items-center justify-center ">
          <div className="w-[760px] mt-4 flex flex-col justify-center items-center gap-5">
            <h1 className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
              Billing Settings
            </h1>
            {children}
          </div>
        </div>
      </PageContent>
    </div>
  );
};

const LegacyUsage: React.FC<{ workspaceId: string }> = async ({ workspaceId }) => {
  const startOfMonth = new Date();
  startOfMonth.setUTCDate(1);
  startOfMonth.setUTCHours(0, 0, 0, 0);

  const year = startOfMonth.getUTCFullYear();
  const month = startOfMonth.getUTCMonth() + 1;

  const [usedVerifications, usedRatelimits] = await Promise.all([
    clickhouse.billing.billableVerifications({
      workspaceId: workspaceId,
      year,
      month,
    }),
    clickhouse.billing.billableRatelimits({
      workspaceId: workspaceId,
      year,
      month,
    }),
  ]);

  return (
    <div className="w-full">
      <SettingCard
        title="Verifications"
        description="Valid key verifications this month."
        border="top"
      >
        <div className="w-full">
          <Input value={format(usedVerifications)} />
        </div>
      </SettingCard>
      <SettingCard title="Ratelimits" description="Valid ratelimits this month." border="bottom">
        <div className="w-full">
          <span className="text-xs text-gray-11">
            <Input value={format(usedRatelimits)} />
          </span>
        </div>
      </SettingCard>
    </div>
  );
};

const Usage: React.FC<{ workspaceId: string; maxRequestsQuota: number }> = async ({
  workspaceId,
  maxRequestsQuota,
}) => {
  const startOfMonth = new Date();
  startOfMonth.setUTCDate(1);
  startOfMonth.setUTCHours(0, 0, 0, 0);

  const year = startOfMonth.getUTCFullYear();
  const month = startOfMonth.getUTCMonth() + 1;

  const [usedVerifications, usedRatelimits] = await Promise.all([
    clickhouse.billing.billableVerifications({
      workspaceId: workspaceId,
      year,
      month,
    }),
    clickhouse.billing.billableRatelimits({
      workspaceId: workspaceId,
      year,
      month,
    }),
  ]);

  const value = usedVerifications + usedRatelimits;

  return (
    <SettingCard
      title="Requests"
      description="Valid key verifications and ratelimits."
      border="both"
    >
      <div className="w-full flex h-full items-center justify-end gap-4">
        <p className="text-sm font-semibold text-gray-12">
          {format(value)} / {format(maxRequestsQuota)} (
          {Math.round((value / maxRequestsQuota) * 100)}%)
        </p>

        <ProgressCircle max={maxRequestsQuota} value={value} />
      </div>
    </SettingCard>
  );
};

function format(n: number): string {
  return Intl.NumberFormat(undefined, { notation: "compact" }).format(n);
}

function clamp(min: number, value: number, max: number): number {
  return Math.min(max, Math.max(value, min));
}

const ProgressCircle: React.FC<{ value: number; max: number }> = ({ value, max }) => {
  const safeValue = clamp(0, value, max);
  const radius = 12;
  const strokeWidth = 3;
  const normalizedRadius = radius - strokeWidth / 2;
  const circumference = normalizedRadius * 2 * Math.PI;
  const offset = circumference - (safeValue / max) * circumference;
  return (
    <>
      <div className="relative flex items-center justify-center">
        <svg
          width={radius * 2}
          height={radius * 2}
          viewBox={`0 0 ${radius * 2} ${radius * 2}`}
          className="-rotate-90 transform"
          aria-label="progress bar"
          aria-valuenow={value}
          aria-valuemax={max}
          data-max={max}
          data-value={safeValue ?? null}
        >
          <circle
            r={normalizedRadius}
            cx={radius}
            cy={radius}
            strokeWidth={strokeWidth}
            fill="transparent"
            stroke=""
            strokeLinecap="round"
            className="transition-colors ease-linear stroke-gray-4"
          />
          {safeValue >= 0 ? (
            <circle
              r={normalizedRadius}
              cx={radius}
              cy={radius}
              strokeWidth={strokeWidth}
              strokeDasharray={`${circumference} ${circumference}`}
              strokeDashoffset={offset}
              fill="transparent"
              stroke=""
              strokeLinecap="round"
              className="stroke-accent-12 transform-gpu transition-all duration-300 ease-in-out"
            />
          ) : null}
        </svg>
      </div>
    </>
  );
};
