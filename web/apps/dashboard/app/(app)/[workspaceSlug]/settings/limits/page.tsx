"use client";

import { PageLoading } from "@/components/dashboard/page-loading";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { queryKeys } from "@/lib/query-keys";
import { SUPPORT_MAILTO } from "@/lib/support";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useQuery } from "@tanstack/react-query";
import { IconCubeOutline18, IconLayers3Outline18, IconNodesOutline18 } from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
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
import { type GroupKey, type LimitGroup, breachedKeys, buildLimitGroups } from "./limit-groups";
import { LimitItem } from "./limit-item";

const CHIPS: Record<GroupKey, { icon: ReactNode; className: string }> = {
  api: { icon: <IconNodesOutline18 />, className: "bg-infoA-3 text-info-11" },
  logs: { icon: <IconLayers3Outline18 />, className: "bg-grayA-3 text-gray-11" },
  compute: { icon: <IconCubeOutline18 />, className: "bg-orangeA-3 text-orange-11" },
};

export default function LimitsPage() {
  const billingUpgrades = useBillingUIUpgrades();
  const limits = useQuery({
    queryKey: queryKeys.workspace.limits,
    queryFn: async () => (await getUnkeyClient().workspace.getLimits()).data,
    enabled: billingUpgrades,
    retry: 1,
  });

  if (!billingUpgrades) {
    notFound();
  }

  if (limits.isLoading) {
    return (
      <Shell>
        <PageLoading message="Loading limits..." />
      </Shell>
    );
  }

  if (!limits.data) {
    return (
      <Shell>
        <EmptyState>
          <EmptyStateHeader>
            <EmptyStateTitle>Limits unavailable</EmptyStateTitle>
            <EmptyStateDescription>
              We could not read the limits for this workspace. Please try again later.
            </EmptyStateDescription>
          </EmptyStateHeader>
        </EmptyState>
      </Shell>
    );
  }

  const groups = buildLimitGroups(limits.data);
  const breached = breachedKeys(groups);

  return (
    <Shell>
      {breached.length > 0 ? <BreachBanner breached={breached} /> : null}
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
