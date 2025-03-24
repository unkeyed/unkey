"use client";
import { WorkspaceSwitcher } from "@/components/navigation/sidebar/team-switcher";
import { UserButton } from "@/components/navigation/sidebar/user-button";
import {
  type NavItem,
  createWorkspaceNavigation,
  resourcesNavigation,
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
import type { Workspace } from "@/lib/db";
import { cn } from "@/lib/utils";
import { SidebarLeftHide, SidebarLeftShow } from "@unkey/icons";
import { useSelectedLayoutSegments } from "next/navigation";
import { useMemo } from "react";
import { NavItems } from "./components/nav-items";
import { ToggleSidebarButton } from "./components/nav-items/toggle-sidebar-button";
import { useWorkspaceNavigation } from "./hooks/use-api-nav-items";

export function AppSidebar({
  ...props
}: React.ComponentProps<typeof Sidebar> & { workspace: Workspace }) {
  const segments = useSelectedLayoutSegments() ?? [];

  const baseNavItems = useMemo(
    () => createWorkspaceNavigation(props.workspace, segments),
    [props.workspace, segments],
  );

  const { enhancedNavItems, loadMore } = useWorkspaceNavigation(baseNavItems);

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

  const { state, isMobile, toggleSidebar } = useSidebar();
  const isCollapsed = state === "collapsed";

  const headerContent = useMemo(
    () => (
      <div
        className={cn(
          "flex w-full",
          isCollapsed ? "justify-center" : "items-center justify-between gap-4",
        )}
      >
        <WorkspaceSwitcher workspace={props.workspace} />
        {state !== "collapsed" && !isMobile && (
          <button type="button" onClick={toggleSidebar}>
            <SidebarLeftHide className="text-gray-8" size="xl-medium" />
          </button>
        )}
      </div>
    ),
    [isCollapsed, props.workspace, state, isMobile, toggleSidebar],
  );

  const resourceNavItems = useMemo(() => resourcesNavigation, []);

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader className="px-4 mb-1 items-center pt-4">{headerContent}</SidebarHeader>
      <SidebarContent className="px-2">
        <SidebarGroup>
          <SidebarMenu className="gap-2">
            {/* Toggle button as NavItem */}
            {state === "collapsed" && (
              <ToggleSidebarButton toggleNavItem={toggleNavItem} toggleSidebar={toggleSidebar} />
            )}
            {enhancedNavItems.map((item) => (
              <NavItems key={item.label} item={item} onLoadMore={loadMore} />
            ))}
            {resourceNavItems.map((item) => (
              <NavItems key={item.label} item={item} />
            ))}
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter className={cn("px-4", !isMobile && "items-center")}>
        <UserButton />
      </SidebarFooter>
    </Sidebar>
  );
}
