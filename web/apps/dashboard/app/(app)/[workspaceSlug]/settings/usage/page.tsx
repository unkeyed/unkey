"use client";

import { PageLoading } from "@/components/dashboard/page-loading";
import { useWorkspaceUsage } from "@/hooks/use-workspace-usage";
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { notFound } from "next/navigation";
import { parseAsString, useQueryState } from "nuqs";
import { type ReactNode, useState } from "react";
import { PlansScreen } from "../billing/components/plans-screen";
import { ApiCard } from "./api-card";
import { ComputeCard, ComputeCardShell, ComputeCardSkeleton } from "./compute-card";
import { breakdownFromUsage, buildComputeTree } from "./compute-tree";
import {
  type UsagePeriod,
  type UsagePeriodOption,
  getUsagePeriodOptions,
  resolveUsagePeriod,
} from "./period";

const ACTIVE_SUBSCRIPTION_STATES = ["active", "trialing", "past_due"];

export default function UsagePage() {
  const billingUpgrades = useBillingUIUpgrades();
  const { workspace, limits, isLoading } = useWorkspace();
  const hasComputePlan = Boolean(workspace?.deployPlan) || Boolean(workspace?.deployPlanOverride);
  const periodOptions = getUsagePeriodOptions(new Date());
  const [periodValue, setPeriodValue] = useQueryState("period", parseAsString);
  const period = resolveUsagePeriod(periodValue);

  const usage = useWorkspaceUsage(period, { enabled: Boolean(workspace) && billingUpgrades });
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
      <Shell options={periodOptions} period={period} onPeriodChange={setPeriodValue}>
        <PageLoading message="Loading usage..." />
      </Shell>
    );
  }

  if (!workspace) {
    notFound();
  }

  const computeTree =
    usage.data === undefined ? undefined : buildComputeTree(breakdownFromUsage(usage.data.compute));
  const compute = hasComputePlan ? (
    usage.isError ? (
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
      verifications={usage.data?.api.verifications ?? null}
      ratelimits={usage.data?.api.ratelimits ?? null}
      quota={limits?.apiBillableOperationsCountMaxPerMonth ?? null}
      feeCents={feeCents}
      isLoading={
        (usage.data === undefined && !usage.isError) || (!planKnown && !billingInfo.isError)
      }
    />
  );

  return (
    <Shell options={periodOptions} period={period} onPeriodChange={setPeriodValue}>
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
  options,
  period,
  onPeriodChange,
}: {
  children: ReactNode;
  options: UsagePeriodOption[];
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
          <Select
            value={period}
            items={options}
            onValueChange={(value) => (value === null ? undefined : onPeriodChange(value))}
          >
            <SelectTrigger
              aria-label="Usage period"
              className="h-8 text-xs"
              wrapperClassName="w-40"
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent align="end" className="bg-background">
              {options.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
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
