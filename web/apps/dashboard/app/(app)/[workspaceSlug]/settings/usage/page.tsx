"use client";

import { PageLoading } from "@/components/dashboard/page-loading";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { formatPeriod } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import {
  Button,
  Empty,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import Link from "next/link";
import { notFound } from "next/navigation";
import type { ReactNode } from "react";
import { ApiCard } from "./api-card";
import { ComputeCard, ComputeCardShell, ComputeCardSkeleton } from "./compute-card";
import { buildComputeTree } from "./compute-tree";

const ACTIVE_SUBSCRIPTION_STATES = ["active", "trialing", "past_due"];

function currentPeriod(): string {
  const now = new Date();
  const monthStartMillis = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1);
  return formatPeriod(monthStartMillis, now.getTime());
}

export default function UsagePage() {
  const billingUpgrades = useBillingUIUpgrades();
  const { workspace, limits, isLoading } = useWorkspace();
  const hasComputePlan = Boolean(workspace?.deployPlan) || Boolean(workspace?.deployPlanOverride);

  const breakdown = trpc.billing.queryDeployUsageBreakdown.useQuery(undefined, {
    enabled: Boolean(workspace) && billingUpgrades && hasComputePlan,
    trpc: { context: { skipBatch: true } },
    retry: 1,
  });
  const apiUsage = trpc.billing.queryUsage.useQuery(undefined, {
    enabled: Boolean(workspace) && billingUpgrades,
    trpc: { context: { skipBatch: true } },
    retry: 1,
  });
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
      <Shell>
        <PageLoading message="Loading usage..." />
      </Shell>
    );
  }

  if (!workspace) {
    notFound();
  }

  const compute = hasComputePlan ? (
    breakdown.isError ? (
      <ComputeCardShell description="Usage per project this period">
        <div className="px-4 py-8">
          <Empty className="w-full">
            <Empty.Title>Compute usage unavailable</Empty.Title>
            <Empty.Description>
              We could not read the Compute breakdown for this period. Please try again later.
            </Empty.Description>
          </Empty>
        </div>
      </ComputeCardShell>
    ) : breakdown.data === undefined ? (
      <ComputeCardSkeleton />
    ) : (
      <ComputeCard tree={buildComputeTree(breakdown.data)} />
    )
  ) : (
    <NoComputePlan workspaceSlug={workspace.slug} />
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
    <Shell>
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

function Shell({ children }: { children: ReactNode }) {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Usage</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <span className="text-[13px] text-gray-10">{currentPeriod()}</span>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>{children}</PageBody>
    </PageContainer>
  );
}

function NoComputePlan({ workspaceSlug }: { workspaceSlug: string }) {
  return (
    <ComputeCardShell description="Usage per app and environment this period">
      <div className="px-4 py-8">
        <Empty className="w-full">
          <Empty.Title>No compute plan</Empty.Title>
          <Empty.Description>Pick a plan to deploy your first app.</Empty.Description>
          <Empty.Actions>
            <Button
              variant="primary"
              size="md"
              render={<Link href={routes.settings.billing({ workspaceSlug })} />}
            >
              Go to billing
            </Button>
          </Empty.Actions>
        </Empty>
      </div>
    </ComputeCardShell>
  );
}
