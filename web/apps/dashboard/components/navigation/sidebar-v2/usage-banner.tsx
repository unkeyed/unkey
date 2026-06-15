"use client";

import { ProgressCircle } from "@/app/(app)/[workspaceSlug]/settings/billing/components/usage";
import { getButtonStyles } from "@/components/navigation/sidebar/app-sidebar/components/nav-items/utils";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import type { Route } from "next";
import Link from "next/link";

export function UsageBanner() {
  const workspace = useWorkspaceNavigation();
  const { quotas } = useWorkspace();
  const { state } = useSidebar();
  const collapsed = state === "collapsed";

  const usage = trpc.billing.queryUsage.useQuery(undefined, {
    refetchOnMount: true,
    refetchInterval: 60 * 1000,
    // Skip batching to prevent analytics slowdown from blocking core UI
    trpc: {
      context: {
        skipBatch: true,
      },
    },
    retry: 1,
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
  const href = `/${workspace.slug}/settings/billing` as Route;

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton asChild tooltip="Usage" className={getButtonStyles(false)}>
          <Link href={href}>
            <ProgressCircle
              value={current}
              max={max}
              color={shouldUpgrade ? "#DD4527" : "#0A9B8B"}
            />
            <span>Usage {Math.round(percentage).toLocaleString()}%</span>
            {shouldUpgrade && !collapsed ? (
              <div className="ml-auto inline-flex h-7 items-center justify-center rounded-md border border-grayA-4 bg-accent-12 px-2 text-sm font-medium text-white drop-shadow-button dark:text-black">
                Upgrade
              </div>
            ) : null}
          </Link>
        </SidebarMenuButton>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
