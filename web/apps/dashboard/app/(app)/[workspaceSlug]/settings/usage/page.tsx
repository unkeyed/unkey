"use client";

import { PageLoading } from "@/components/dashboard/page-loading";
import { useFlag } from "@/lib/flags/provider";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import {
  Button,
  Empty,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";
import Link from "next/link";
import { notFound } from "next/navigation";
import type { ReactNode } from "react";
import { FREE_TIER_QUOTA } from "../billing/components/constants";
import { ApiCard } from "./api-card";
import { ComputeCard, ComputeCardShell, ComputeCardSkeleton } from "./compute-card";
import { buildComputeTree } from "./compute-tree";

const ACTIVE_SUBSCRIPTION_STATES = ["active", "trialing", "past_due"];

/** Both endpoints aggregate over the UTC calendar month, so the period label does too. */
function currentPeriod(): string {
  return new Date().toLocaleDateString("en-US", {
    month: "long",
    year: "numeric",
    timeZone: "UTC",
  });
}

export default function UsagePage() {
  const deployBillingEnabled = useFlag("deployBilling");
  const { workspace, isLoading } = useWorkspace();
  const hasComputePlan = Boolean(workspace?.deployPlan) || Boolean(workspace?.deployPlanOverride);

  const breakdown = trpc.billing.queryDeployUsageBreakdown.useQuery(undefined, {
    enabled: Boolean(workspace) && hasComputePlan,
    trpc: { context: { skipBatch: true } },
    retry: 1,
  });
  const deployUsage = trpc.billing.queryDeployUsage.useQuery(undefined, {
    enabled: Boolean(workspace) && hasComputePlan,
    trpc: { context: { skipBatch: true } },
    retry: 1,
  });
  const apiUsage = trpc.billing.queryUsage.useQuery(undefined, {
    enabled: Boolean(workspace),
    trpc: { context: { skipBatch: true } },
    retry: 1,
  });
  const billingInfo = trpc.stripe.getBillingInfo.useQuery(undefined, {
    enabled: Boolean(workspace),
    staleTime: 30_000,
    retry: 1,
  });

  if (!deployBillingEnabled) {
    notFound();
  }

  if (isLoading || !workspace) {
    return (
      <Shell>
        <PageLoading message="Loading usage..." />
      </Shell>
    );
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
      <ComputeCard
        tree={buildComputeTree(breakdown.data)}
        workspaceCents={deployUsage.data?.grossCents ?? null}
      />
    )
  ) : (
    <NoComputePlan workspaceSlug={workspace.slug} />
  );

  const info = billingInfo.data;
  const apiProduct =
    info?.subscription &&
    info.currentProductId &&
    ACTIVE_SUBSCRIPTION_STATES.includes(info.subscription.status)
      ? info.products.find((product) => product.id === info.currentProductId)
      : undefined;
  const planKnown = info !== undefined;

  const api = (
    <ApiCard
      verifications={apiUsage.data?.billableVerifications ?? null}
      ratelimits={apiUsage.data?.billableRatelimits ?? null}
      quota={planKnown ? (apiProduct?.quotas.requestsPerMonth ?? FREE_TIER_QUOTA) : null}
      feeCents={planKnown ? (apiProduct?.dollar ?? 0) * 100 : null}
      isLoading={
        (apiUsage.data === undefined && !apiUsage.isError) || (!planKnown && !billingInfo.isError)
      }
    />
  );

  // The product being paid for leads. Without a Compute plan there is nothing to
  // break down, so its empty state sits below API management rather than heading
  // the page.
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
          <PageHeaderDescription>
            Gross usage for {currentPeriod()}. Credits and invoices are on Billing.
          </PageHeaderDescription>
        </PageHeaderContent>
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
