"use client";

import { PageLoading } from "@/components/dashboard/page-loading";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { routes } from "@/lib/navigation/routes";
import { SUPPORT_MAILTO } from "@/lib/support";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import { Cube, Layers3, Link4, Nodes } from "@unkey/icons";
import {
  Button,
  Empty,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemHeader,
  ItemMedia,
  ItemSeparator,
  ItemTitle,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Fragment, type ReactNode } from "react";
import { BreachBanner } from "./breach-banner";
import {
  type GroupKey,
  type LimitGroup,
  type Measured,
  breachedGroups,
  buildLimitGroups,
} from "./limit-groups";
import { LimitItem } from "./limit-item";

const CHIPS: Record<GroupKey, { icon: ReactNode; className: string }> = {
  api: { icon: <Nodes />, className: "bg-infoA-3 text-info-11" },
  logs: { icon: <Layers3 />, className: "bg-grayA-3 text-gray-11" },
  compute: { icon: <Cube />, className: "bg-orangeA-3 text-orange-11" },
  domains: { icon: <Link4 />, className: "bg-grayA-3 text-gray-11" },
};

function measured<T>(query: { data: T | undefined; isError: boolean }): Measured<T> {
  if (query.isError) {
    return { state: "error" };
  }
  return query.data === undefined ? { state: "loading" } : { state: "ready", value: query.data };
}

export default function LimitsPage() {
  const billingUpgrades = useBillingUIUpgrades();
  const { workspace, limits, isLoading } = useWorkspace();
  const hasComputePlan = Boolean(workspace?.deployPlan) || Boolean(workspace?.deployPlanOverride);

  const usage = trpc.billing.queryUsage.useQuery(undefined, {
    enabled: Boolean(workspace) && billingUpgrades,
    trpc: { context: { skipBatch: true } },
    retry: 1,
  });
  const allocation = trpc.billing.queryComputeAllocation.useQuery(undefined, {
    enabled: Boolean(workspace) && billingUpgrades && hasComputePlan,
    trpc: { context: { skipBatch: true } },
    retry: 1,
  });
  const customDomains = trpc.deploy.customDomain.count.useQuery(undefined, {
    enabled: Boolean(workspace) && billingUpgrades,
    trpc: { context: { skipBatch: true } },
    retry: 1,
  });

  if (!billingUpgrades) {
    notFound();
  }

  if (isLoading) {
    return (
      <Shell>
        <PageLoading message="Loading limits..." />
      </Shell>
    );
  }

  if (!limits || !workspace) {
    return (
      <Shell>
        <Empty>
          <Empty.Title>Limits unavailable</Empty.Title>
          <Empty.Description>
            We could not read the limits for this workspace. Please try again later.
          </Empty.Description>
        </Empty>
      </Shell>
    );
  }

  const groups = buildLimitGroups({
    limits,
    hasComputePlan,
    apiOperations: measured({ data: usage.data?.billableTotal, isError: usage.isError }),
    allocation: measured(allocation),
    customDomains: measured(customDomains),
  });
  const breached = breachedGroups(groups);

  return (
    <Shell>
      {breached.length > 0 ? (
        <BreachBanner
          breached={breached}
          billingHref={routes.settings.billing({ workspaceSlug: workspace.slug, intent: "api" })}
        />
      ) : null}
      {groups.map((group) => (
        <Group key={group.key} group={group} />
      ))}
    </Shell>
  );
}

function Shell({ children }: { children: ReactNode }) {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Limits</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button variant="primary" render={<Link href={SUPPORT_MAILTO} />}>
            Request a change
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>{children}</PageBody>
    </PageContainer>
  );
}

function Group({ group }: { group: LimitGroup }) {
  const chip = CHIPS[group.key];

  return (
    <ItemGroup variant="outline">
      <ItemHeader>
        <ItemMedia className={chip.className}>{chip.icon}</ItemMedia>
        <ItemContent>
          <ItemTitle>{group.title}</ItemTitle>
          <ItemDescription>{group.description}</ItemDescription>
        </ItemContent>
      </ItemHeader>
      {group.rows.map((row) => (
        <Fragment key={row.name}>
          <ItemSeparator />
          <LimitItem row={row} />
        </Fragment>
      ))}
    </ItemGroup>
  );
}
