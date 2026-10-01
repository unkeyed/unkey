"use client";

import { PageLoading } from "@/components/dashboard/page-loading";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
  Tabs,
  TabsList,
  TabsTrigger,
} from "@unkey/ui";
import { notFound } from "next/navigation";
import { parseAsString, useQueryState } from "nuqs";
import { type ReactNode, useMemo, useState } from "react";
import { PlansScreen } from "../billing/components/plans-screen";
import { ApiCard } from "./api-card";
import { ComputeCard, ComputeCardShell, ComputeCardSkeleton } from "./compute-card";
import { buildComputeTree } from "./compute-tree";
import { type UsagePeriod, getUsagePeriods, resolveUsagePeriod } from "./period";

const ACTIVE_SUBSCRIPTION_STATES = ["active", "trialing", "past_due"];

export default function UsagePage() {
  const billingUpgrades = useBillingUIUpgrades();
  const { workspace, limits, isLoading } = useWorkspace();
  const hasComputePlan = Boolean(workspace?.deployPlan) || Boolean(workspace?.deployPlanOverride);
  const now = useMemo(() => new Date(), []);
  const periods = useMemo(() => getUsagePeriods(now), [now]);
  const [periodValue, setPeriodValue] = useQueryState("period", parseAsString);
  const period = resolveUsagePeriod(periodValue, periods);

  const breakdown = trpc.billing.queryDeployUsageBreakdown.useQuery(
    { monthsAgo: period.monthsAgo },
    {
      enabled: Boolean(workspace) && billingUpgrades && hasComputePlan,
      trpc: { context: { skipBatch: true } },
      retry: 1,
    },
  );
  const apiUsage = trpc.billing.queryUsage.useQuery(
    { monthsAgo: period.monthsAgo },
    {
      enabled: Boolean(workspace) && billingUpgrades,
      trpc: { context: { skipBatch: true } },
      retry: 1,
    },
  );
  const billingInfo = trpc.stripe.getBillingInfo.useQuery(undefined, {
    enabled: Boolean(workspace) && billingUpgrades,
    staleTime: 30_000,
    retry: 1,
  });

  if (!billingUpgrades) {
    notFound();
  }

  if (isLoading) {
    return (
      <Shell periods={periods} period={period} onPeriodChange={setPeriodValue}>
        <PageLoading message="Loading usage..." />
      </Shell>
    );
  }

  if (!workspace) {
    notFound();
  }

  const computeTree = breakdown.data === undefined ? undefined : buildComputeTree(breakdown.data);
  const compute = hasComputePlan ? (
    breakdown.isError ? (
      <ComputeCardShell description="Usage per project this period">
        <div className="px-4 py-8">
          <EmptyState frame="none">
            <EmptyStateHeader>
              <EmptyStateTitle>Compute usage unavailable</EmptyStateTitle>
              <EmptyStateDescription>
                We could not read the Compute breakdown for this period. Please try again later.
              </EmptyStateDescription>
            </EmptyStateHeader>
          </EmptyState>
        </div>
      </ComputeCardShell>
    ) : computeTree === undefined ? (
      <ComputeCardSkeleton />
    ) : (
      <ComputeCard tree={computeTree} period={period} />
    )
  ) : (
    <NoComputePlan />
  );

  const info = billingInfo.data;
  const planKnown = info !== undefined;
  const hasActiveSubscription = Boolean(
    info?.subscription && ACTIVE_SUBSCRIPTION_STATES.includes(info.subscription.status),
  );
  const apiProduct = hasActiveSubscription
    ? info?.products.find((product) => product.id === info.currentProductId)
    : undefined;

  let feeCents: number | null;
  if (!planKnown) {
    feeCents = null;
  } else if (!hasActiveSubscription) {
    feeCents = 0;
  } else if (apiProduct === undefined) {
    feeCents = null;
  } else {
    feeCents = apiProduct.dollar * 100;
  }

  // The quota comes from the workspace's resolved limits row, the same source the
  // Limits page reads. Deriving it from the Stripe catalog would print a different
  // number on two adjacent pages for any workspace with an overridden limit.
  const api = (
    <ApiCard
      verifications={apiUsage.data?.billableVerifications ?? null}
      ratelimits={apiUsage.data?.billableRatelimits ?? null}
      quota={limits?.apiBillableOperationsCountMaxPerMonth ?? null}
      feeCents={feeCents}
      isLoading={
        (apiUsage.data === undefined && !apiUsage.isError) || (!planKnown && !billingInfo.isError)
      }
    />
  );

  return (
    <Shell periods={periods} period={period} onPeriodChange={setPeriodValue}>
      {hasComputePlan ? (
        <>
          {compute}
          {api}
        </>
      ) : (
        <>
          {api}
          {compute}
        </>
      )}
    </Shell>
  );
}

function Shell({
  children,
  periods,
  period,
  onPeriodChange,
}: {
  children: ReactNode;
  periods: UsagePeriod[];
  period: UsagePeriod;
  onPeriodChange: (value: string) => Promise<URLSearchParams>;
}) {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Usage</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <Tabs value={period.value} onValueChange={(value) => onPeriodChange(value)}>
            <TabsList aria-label="Usage period" className="h-8 gap-0.5 border bg-raised p-0.5">
              {periods.map((option) => (
                <TabsTrigger
                  key={option.value}
                  value={option.value}
                  className="h-full px-2.5 py-0 text-xs data-active:bg-grayA-3 data-active:shadow-none"
                >
                  {option.label}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>{children}</PageBody>
    </PageContainer>
  );
}

function NoComputePlan() {
  const [plansOpen, setPlansOpen] = useState(false);
  return (
    <ComputeCardShell description="Usage per app and environment this period">
      <div className="px-4 py-8">
        <EmptyState frame="none">
          <EmptyStateHeader>
            <EmptyStateTitle>No compute plan</EmptyStateTitle>
            <EmptyStateDescription>Pick a plan to deploy your first app.</EmptyStateDescription>
          </EmptyStateHeader>
          <EmptyStateActions>
            <Button variant="primary" size="md" onClick={() => setPlansOpen(true)}>
              Choose a plan
            </Button>
          </EmptyStateActions>
        </EmptyState>
        <PlansScreen open={plansOpen} onOpenChange={setPlansOpen} reason="deploy" />
      </div>
    </ComputeCardShell>
  );
}
