"use client";

import { Sidebar, SidebarContent, SidebarFooter, useSidebar } from "@/components/ui/sidebar";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import { IconSidebarLeftHideOutline18, IconSidebarLeftShowOutline18 } from "nucleo-ui-outline-18";
import { SidebarBody } from "./sidebar-body";
import { UsageBanner } from "./usage-banner";
import { UsageCard } from "./usage-card";

export const SIDEBAR_WIDTH_VARS: React.CSSProperties & {
  "--sidebar-width": string;
  "--sidebar-width-icon": string;
} = {
  "--sidebar-width": "13rem",
  "--sidebar-width-icon": "3rem",
};

type Props = React.ComponentProps<typeof Sidebar>;

export function SidebarV2(props: Props) {
  const { isMobile } = useSidebar();
  const billingUpgrades = useBillingUIUpgrades();
  if (isMobile) {
    return null;
  }
  return (
    <Sidebar
      {...props}
      collapsible="icon"
      // absolute, not the default viewport-fixed: the layout's relative content
      // row already starts below the top nav and the paused banner.
      className={cn("absolute h-auto [&_[data-sidebar=sidebar]]:bg-gray-1", props.className)}
    >
      <SidebarContent>
        <SidebarBody />
      </SidebarContent>
      <SidebarFooter className="mx-0 gap-2 border-t-0 p-2">
        {billingUpgrades ? <UsageCard /> : <UsageBanner />}
        <CollapseButton />
      </SidebarFooter>
    </Sidebar>
  );
}

function CollapseButton() {
  const { state, toggleSidebar } = useSidebar();
  const collapsed = state === "collapsed";
  const Icon = collapsed ? IconSidebarLeftShowOutline18 : IconSidebarLeftHideOutline18;
  const label = collapsed ? "Expand sidebar" : "Collapse sidebar";
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <button
            type="button"
            onClick={toggleSidebar}
            aria-label={label}
            className="flex size-8 items-center justify-center rounded-md text-gray-11 hover:bg-grayA-3 hover:text-gray-12"
          >
            <Icon className="size-4 shrink-0" />
          </button>
        }
      />
      <TooltipContent
        side="right"
        align="center"
        className="dark:bg-white bg-black text-gray-1 px-2 py-1 border border-accent-6 shadow-md font-normal text-xs"
      >
        {label}
      </TooltipContent>
    </Tooltip>
  );
}
