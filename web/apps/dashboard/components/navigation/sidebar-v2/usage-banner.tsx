"use client";

import { ProgressCircle } from "@/app/(app)/[workspaceSlug]/settings/billing/components/usage";
import { getButtonStyles } from "@/components/navigation/sidebar/app-sidebar/components/nav-items/utils";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar";
import { useWorkspaceLimits } from "@/hooks/use-workspace-limits";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { Skeleton } from "@unkey/ui";
import Link from "next/link";

export function UsageBanner() {
  const workspace = useWorkspaceNavigation();
  const { state } = useSidebar();
  const collapsed = state === "collapsed";

  const workspaceLimits = useWorkspaceLimits({ refetchInterval: 60 * 1000 });

  if (workspaceLimits.isError) {
    return null;
  }

  const billable = workspaceLimits.data?.api.billableOperations;
  if (billable !== undefined && billable.limit <= 0) {
    console.error(
      "UsageBanner: billableOperations.limit must be greater than 0, got:",
      billable.limit,
    );
    return null;
  }

  const current = billable?.used ?? 0;
  const max = billable?.limit ?? 1;
  const percentage = (current / max) * 100;
  const shouldUpgrade = percentage > 90;
  const href = routes.settings.billing({ workspaceSlug: workspace.slug });

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton
          tooltip="Usage"
          className={getButtonStyles(false)}
          render={
            <Link href={href}>
              <ProgressCircle
                value={current}
                max={max}
                color={shouldUpgrade ? "#DD4527" : "#0A9B8B"}
              />
              <span>
                Usage{" "}
                {billable === undefined ? (
                  <Skeleton className="inline-block h-3 w-7 align-middle" />
                ) : (
                  `${Math.round(percentage).toLocaleString()}%`
                )}
              </span>
              {shouldUpgrade && !collapsed ? (
                <div className="ml-auto inline-flex h-7 items-center justify-center rounded-md border bg-gray-12 px-2 text-sm font-medium text-white dark:text-black">
                  Upgrade
                </div>
              ) : null}
            </Link>
          }
        />
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
