"use client";
import { WorkspaceSwitcher } from "@/components/navigation/sidebar/team-switcher";
import { UserButton } from "@/components/navigation/sidebar/user-button";
import {
  type NavItem,
  createWorkspaceNavigation,
} from "@/components/navigation/sidebar/workspace-navigations";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarHeader,
  SidebarMenu,
  useSidebar,
} from "@/components/ui/sidebar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Quotas, Workspace } from "@/lib/db";
import { cn } from "@/lib/utils";
import { SidebarLeftHide, SidebarLeftShow } from "@unkey/icons";
import { useRouter, useSelectedLayoutSegments } from "next/navigation";
import { useEffect, useMemo } from "react";
import { HelpButton } from "../help-button";
import { UsageBanner } from "../usage-banner";
import { NavItems } from "./components/nav-items";
import { ToggleSidebarButton } from "./components/nav-items/toggle-sidebar-button";
import { useApiNavigation } from "./hooks/use-api-navigation";
import { useProjectNavigation } from "./hooks/use-projects-navigation";
import { useRatelimitNavigation } from "./hooks/use-ratelimit-navigation";

export function AppSidebar({
  ...props
}: React.ComponentProps<typeof Sidebar> & {
  workspace: Workspace & { quotas: Quotas | null };
}) {
  const segments = useSelectedLayoutSegments() ?? [];
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  // Refresh the router when workspace changes to update sidebar
  useEffect(() => {
    if (!props.workspace.id) {
      return;
    }
    router.refresh();
  }, [router, props.workspace.id]);

  // Create base navigation items
  const baseNavItems = useMemo(
    () => createWorkspaceNavigation(segments, workspace),
    [segments, workspace],
  );

  const { enhancedNavItems: apiAddedNavItems } = useApiNavigation(baseNavItems);

  const { enhancedNavItems: ratelimitAddedNavItems } = useRatelimitNavigation(apiAddedNavItems);

  const { enhancedNavItems: projectAddedNavItems } = useProjectNavigation(ratelimitAddedNavItems);

  const toggleNavItem: NavItem = useMemo(
    () => ({
      label: "Toggle Sidebar",
      href: "#",
      icon: SidebarLeftShow,
      active: false,
      tooltip: "Toggle Sidebar",
    }),
    [],
  );

  const { state, isMobile, toggleSidebar, openMobile } = useSidebar();
  const isCollapsed = state === "collapsed";

  const headerContent = useMemo(
    () => (
      <div
        className={cn(
          "flex w-full",
          isCollapsed ? "justify-center" : "items-center justify-between gap-4",
        )}
      >
        <WorkspaceSwitcher />
        {state !== "collapsed" && !isMobile && (
          <button type="button" onClick={toggleSidebar}>
            <SidebarLeftHide className="text-gray-8" iconSize="xl-medium" />
          </button>
        )}
      </div>
    ),
    [isCollapsed, state, isMobile, toggleSidebar],
  );

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader className="px-4 items-center pt-4">{headerContent}</SidebarHeader>
      <SidebarContent className="px-2 flex flex-col justify-between">
        <SidebarGroup>
          <SidebarMenu className="gap-2">
            {state === "collapsed" && (
              <ToggleSidebarButton toggleNavItem={toggleNavItem} toggleSidebar={toggleSidebar} />
            )}

            {projectAddedNavItems.map((item) => (
              <NavItems key={item.label as string} item={item} />
            ))}
          </SidebarMenu>
        </SidebarGroup>

        <SidebarGroup>
          <UsageBanner quotas={props.workspace.quotas} />
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <div
          className={cn("flex items-center justify-between gap-2", {
            "flex-col-reverse": state === "collapsed",
            "flex-row": state === "expanded",
          })}
        >
          <UserButton
            isCollapsed={(state === "collapsed" || isMobile) && !(isMobile && openMobile)}
            isMobile={isMobile}
            isMobileSidebarOpen={openMobile}
          />

          <HelpButton />
        </div>
      </SidebarFooter>
    </Sidebar>
  );
}
