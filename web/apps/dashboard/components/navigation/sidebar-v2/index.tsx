"use client";

import { Sidebar, SidebarContent, SidebarFooter, useSidebar } from "@/components/ui/sidebar";
import { cn } from "@/lib/utils";
import { SidebarLeftHide, SidebarLeftShow } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import { TOP_NAV_HEIGHT } from "../top-nav";
import { SidebarBody } from "./sidebar-body";
import { UsageBanner } from "./usage-banner";

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
  if (isMobile) {
    return null;
  }
  return (
    <Sidebar
      {...props}
      collapsible="icon"
      className={cn("[&_[data-sidebar=sidebar]]:bg-gray-1", props.className)}
      style={{
        top: TOP_NAV_HEIGHT,
        height: `calc(100svh - ${TOP_NAV_HEIGHT}px)`,
      }}
    >
      <SidebarContent>
        <SidebarBody />
      </SidebarContent>
      <SidebarFooter className="mx-0 gap-2 border-t-0 p-2">
        <UsageBanner />
        <CollapseButton />
      </SidebarFooter>
    </Sidebar>
  );
}

function CollapseButton() {
  const { state, toggleSidebar } = useSidebar();
  const collapsed = state === "collapsed";
  const Icon = collapsed ? SidebarLeftShow : SidebarLeftHide;
  const label = collapsed ? "Expand sidebar" : "Collapse sidebar";
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          onClick={toggleSidebar}
          aria-label={label}
          className="flex size-8 items-center justify-center rounded-md text-gray-11 hover:bg-grayA-3 hover:text-gray-12"
        >
          <Icon iconSize="md-regular" className="shrink-0" />
        </button>
      </TooltipTrigger>
      <TooltipContent
        side="right"
        align="center"
        className="dark:bg-white bg-black text-gray-1 px-2 py-1 border border-accent-6 shadow-md font-medium text-xs"
      >
        {label}
      </TooltipContent>
    </Tooltip>
  );
}
