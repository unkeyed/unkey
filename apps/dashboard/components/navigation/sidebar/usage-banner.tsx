"use client";
import { ProgressCircle } from "@/app/(app)/[workspaceSlug]/settings/billing/components/usage";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import type { Quotas } from "@unkey/db";
import { Button, Loading } from "@unkey/ui";
import Link from "next/link";
import type React from "react";
import { Suspense } from "react";
import { FlatNavItem } from "./app-sidebar/components/nav-items/flat-nav-item";
type Props = {
  quotas: Quotas | null;
};

export const UsageBanner: React.FC<Props> = ({ quotas }) => {
  const workspace = useWorkspaceNavigation();

  const usage = trpc.billing.queryUsage.useQuery(undefined, {
    refetchOnMount: true,
    refetchInterval: 60 * 1000,
  });

  const current = usage.data?.billableTotal ?? 0;
  const max = quotas?.requestsPerMonth;

  if (max === undefined || max === null) {
    console.error("UsageBanner: quotas.requestsPerMonth is undefined or null");
    return null;
  }

  if (max <= 0) {
    console.error("UsageBanner: quotas.requestsPerMonth must be greater than 0, got:", max);
    return null;
  }

  const percentage = (current / max) * 100;
  const shouldUpgrade = percentage > 90;

  return (
    <Suspense fallback={<Loading type="spinner" />}>
      <FlatNavItem
        item={{
          tooltip: "Usage",
          icon: () => (
            <ProgressCircle
              value={current}
              max={max}
              color={shouldUpgrade ? "#DD4527" : "#0A9B8B"}
            />
          ),
          href: `/${workspace.slug}/settings/billing`,
          label: `Usage ${Math.round(percentage).toLocaleString()}%`,
          tag: shouldUpgrade ? (
            <Link href="/settings/billing">
              <Button variant="primary" size="sm">
                Upgrade
              </Button>
            </Link>
          ) : null,
        }}
      />
    </Suspense>
  );
};
