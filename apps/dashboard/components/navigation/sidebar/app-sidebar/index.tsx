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
  SidebarMenuButton,
  useSidebar,
} from "@/components/ui/sidebar";
import type { Quotas, Workspace } from "@/lib/db";
import { cn } from "@/lib/utils";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { ChevronLeft, SidebarLeftHide, SidebarLeftShow } from "@unkey/icons";
import { useRouter, useSelectedLayoutSegments } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { HelpButton } from "../help-button";
import { UsageBanner } from "../usage-banner";
import { NavItems } from "./components/nav-items";
import { ToggleSidebarButton } from "./components/nav-items/toggle-sidebar-button";
import { getButtonStyles } from "./components/nav-items/utils";
import { useApiNavigation } from "./hooks/use-api-navigation";
import { useProjectNavigation } from "./hooks/use-projects-navigation";
import { useRatelimitNavigation } from "./hooks/use-ratelimit-navigation";

// Configuration for solo mode routes
const SOLO_MODE_CONFIG = {
  projects: {
    routeSegment: "projects",
    backButtonText: "Back to main menu",
  },
  // Future solo modes can be added here:
  // apis: {
  //   routeSegment: "apis",
  //   backButtonText: "Back to main menu",
  // }
} as const;

type SoloModeType = keyof typeof SOLO_MODE_CONFIG;

export function AppSidebar({
  ...props
}: React.ComponentProps<typeof Sidebar> & {
  workspace: Workspace & { quotas: Quotas | null };
}) {
  const segments = useSelectedLayoutSegments() ?? [];
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  // Get the current solo mode type based on the route
  const currentSoloModeType = useMemo(() => {
    const firstSegment = segments.at(0) ?? "";
    return Object.entries(SOLO_MODE_CONFIG).find(
      ([_, config]) => config.routeSegment === firstSegment
    )?.[0] as SoloModeType | undefined;
  }, [segments]);

  const [soloModeOverride, setSoloModeOverride] = useState(() =>
    Boolean(currentSoloModeType)
  );

  // Refresh the router when workspace changes to update sidebar
  useEffect(() => {
    if (!props.workspace.id) {
      return;
    }
    router.refresh();
  }, [router, props.workspace.id]);

  // Update solo mode when route changes
  useEffect(() => {
    setSoloModeOverride(Boolean(currentSoloModeType));
  }, [currentSoloModeType]);

  // Create base navigation items
  const baseNavItems = useMemo(
    () => createWorkspaceNavigation(segments, workspace),
    [segments, workspace]
  );

  const { enhancedNavItems: apiAddedNavItems } = useApiNavigation(baseNavItems);

  const { enhancedNavItems: ratelimitAddedNavItems } =
    useRatelimitNavigation(apiAddedNavItems);

  const { enhancedNavItems: projectAddedNavItems } = useProjectNavigation(
    ratelimitAddedNavItems
  );

  const handleToggleCollapse = useCallback((item: NavItem, isOpen: boolean) => {
    // Check if this item corresponds to any solo mode route
    const matchingSoloMode = Object.entries(SOLO_MODE_CONFIG).find(
      ([_, config]) => item.href.includes(`/${config.routeSegment}`)
    );

    if (matchingSoloMode) {
      setSoloModeOverride(isOpen);
    }
  }, []);

  const toggleNavItem: NavItem = useMemo(
    () => ({
      label: "Toggle Sidebar",
      href: "#",
      icon: SidebarLeftShow,
      active: false,
      tooltip: "Toggle Sidebar",
    }),
    []
  );

  const { state, isMobile, toggleSidebar, openMobile } = useSidebar();
  const isCollapsed = state === "collapsed";

  const headerContent = useMemo(
    () => (
      <div
        className={cn(
          "flex w-full",
          isCollapsed ? "justify-center" : "items-center justify-between gap-4"
        )}
      >
        <WorkspaceSwitcher />
        {state !== "collapsed" && !isMobile && (
          <button type="button" onClick={toggleSidebar}>
            <SidebarLeftHide className="text-gray-8" size="xl-medium" />
          </button>
        )}
      </div>
    ),
    [isCollapsed, state, isMobile, toggleSidebar]
  );

  const currentSoloConfig = currentSoloModeType
    ? SOLO_MODE_CONFIG[currentSoloModeType]
    : null;

  const hasSoloActive = Boolean(currentSoloConfig && soloModeOverride);

  // Find the navigation item that corresponds to the current solo mode
  const currentSoloNavItem = useMemo(() => {
    if (!currentSoloConfig) {
      return null;
    }

    return projectAddedNavItems.find((item) =>
      item.href.includes(`/${currentSoloConfig.routeSegment}`)
    );
  }, [currentSoloConfig, projectAddedNavItems]);

  const getForceCollapsedForItem = useCallback(
    (item: NavItem): boolean => {
      if (!currentSoloConfig) {
        return false;
      }

      // Force collapse if this item corresponds to the current solo mode and soloModeOverride is false
      const itemMatchesSoloMode = item.href.includes(
        `/${currentSoloConfig.routeSegment}`
      );
      return itemMatchesSoloMode && !soloModeOverride;
    },
    [currentSoloConfig, soloModeOverride]
  );

  const handleBackToMainMenu = useCallback(() => {
    if (currentSoloNavItem) {
      handleToggleCollapse(currentSoloNavItem, false);
    }
  }, [currentSoloNavItem, handleToggleCollapse]);

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader className="px-4 items-center pt-4">
        {headerContent}
      </SidebarHeader>
      <SidebarContent className="px-2 flex flex-col justify-between">
        <SidebarGroup>
          <SidebarMenu className="gap-2">
            {state === "collapsed" && (
              <ToggleSidebarButton
                toggleNavItem={toggleNavItem}
                toggleSidebar={toggleSidebar}
              />
            )}

            {hasSoloActive && currentSoloConfig && (
              <SidebarMenuButton
                className={getButtonStyles(false, false)}
                onClick={handleBackToMainMenu}
              >
                <div className="flex items-center gap-2 justify-start w-full">
                  <div className="size-5 flex items-center">
                    <ChevronLeft
                      className="shrink-0 !size-3 text-gray-10"
                      size="sm-medium"
                    />
                  </div>
                  <div className="text-accent-9 text-xs leading-6">
                    {currentSoloConfig.backButtonText}
                  </div>
                </div>
              </SidebarMenuButton>
            )}

            {projectAddedNavItems.map((item) => (
              <NavItems
                key={item.label as string}
                item={item}
                onToggleCollapse={handleToggleCollapse}
                forceCollapsed={getForceCollapsedForItem(item)}
                className={cn(
                  hasSoloActive && !item.active ? "hidden" : "block"
                )}
              />
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
            isCollapsed={
              (state === "collapsed" || isMobile) && !(isMobile && openMobile)
            }
            isMobile={isMobile}
            isMobileSidebarOpen={openMobile}
          />

          <HelpButton />
        </div>
      </SidebarFooter>
    </Sidebar>
  );
}
