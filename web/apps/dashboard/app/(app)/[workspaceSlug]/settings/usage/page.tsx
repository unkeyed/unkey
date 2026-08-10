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
import { ApiCard } from "./api-card";
import { ComputeCard } from "./compute-card";
import { buildComputeTree } from "./compute-tree";

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
      <Empty>
        <Empty.Title>Compute usage unavailable</Empty.Title>
        <Empty.Description>
          We could not read the Compute breakdown for this period. Please try again later.
        </Empty.Description>
      </Empty>
    ) : breakdown.data === undefined ? (
      <div className="h-40 w-full animate-pulse rounded-lg bg-grayA-3" />
    ) : (
      <ComputeCard
        tree={buildComputeTree(breakdown.data)}
        workspaceCents={deployUsage.data?.grossCents ?? null}
        activeKeys={deployUsage.data?.activeKeys ?? null}
      />
    )
  ) : (
    <NoComputePlan workspaceSlug={workspace.slug} />
  );

  const api = (
    <ApiCard
      verifications={apiUsage.data?.billableVerifications ?? null}
      ratelimits={apiUsage.data?.billableRatelimits ?? null}
      isLoading={apiUsage.data === undefined && !apiUsage.isError}
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
    <Empty>
      <Empty.Title>No Compute plan</Empty.Title>
      <Empty.Description>
        Pick a plan to deploy your first app. Compute usage appears here once it runs.
      </Empty.Description>
      <Empty.Actions>
        <Button
          variant="primary"
          render={<Link href={routes.settings.billing({ workspaceSlug })} />}
        >
          Go to billing
        </Button>
      </Empty.Actions>
    </Empty>
  );
}
