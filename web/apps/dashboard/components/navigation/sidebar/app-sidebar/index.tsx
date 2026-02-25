"use client";
import { ContextNavigation } from "@/components/navigation/sidebar/context-navigation";
import {
  RESOURCE_TYPE_PLURAL,
  RESOURCE_TYPE_ROUTES,
} from "@/components/navigation/sidebar/navigation-configs";
import { ProductSwitcher } from "@/components/navigation/sidebar/product-switcher";
import { ResourceHeading } from "@/components/navigation/sidebar/resource-heading";
import { WorkspaceSwitcher } from "@/components/navigation/sidebar/team-switcher";
import { WorkspaceSection } from "@/components/navigation/sidebar/workspace-section";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarHeader,
  SidebarMenu,
  useSidebar,
} from "@/components/ui/sidebar";
import { useNavigationContext } from "@/hooks/use-navigation-context";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Quotas, Workspace } from "@/lib/db";
import { cn } from "@/lib/utils";
import { ChevronLeft, SidebarLeftHide, SidebarLeftShow } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { HelpButton } from "../help-button";
import { UsageBanner } from "../usage-banner";
import type { NavItem } from "../workspace-navigations";
import { ToggleSidebarButton } from "./components/nav-items/toggle-sidebar-button";

export function AppSidebar({
  ...props
}: React.ComponentProps<typeof Sidebar> & {
  workspace: Workspace & { quotas: Quotas | null };
}) {
  const router = useRouter();
  const context = useNavigationContext();
  const workspace = useWorkspaceNavigation();
  const [fetchedResourceData, setFetchedResourceData] = useState<{
    name: string | undefined;
    contextKey: string;
  }>();

  // Create a stable key from context to track changes
  const contextKey = useMemo(() => {
    if (context.type === "resource") {
      return `${context.resourceType}-${context.resourceId}`;
    }
    return context.product;
  }, [context]);

  // Callback to receive fetched resource name from ContextNavigation
  const handleResourceNameFetched = useCallback(
    (name: string | undefined) => {
      setFetchedResourceData({ name, contextKey });
    },
    [contextKey],
  );

  // Derive the current resource name - only use fetched name if it matches current context
  const fetchedResourceName =
    fetchedResourceData?.contextKey === contextKey ? fetchedResourceData.name : undefined;

  // Refresh the router when workspace changes to update sidebar
  useEffect(() => {
    if (!props.workspace.id) {
      return;
    }
    router.refresh();
  }, [router, props.workspace.id]);

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

  // Determine if we should show resource heading
  const showResourceHeading = context.type === "resource";

  // Determine which product to show in the switcher
  const currentProduct = useMemo((): "api-management" | "deploy" => {
    switch (context.type) {
      case "product":
        return context.product;
      case "resource": {
        switch (context.resourceType) {
          case "api":
          case "namespace":
            return "api-management";
          case "project":
            return "deploy";
          default: {
            // Exhaustiveness check - will cause compile error if new resource type added
            const _exhaustive: never = context.resourceType;
            return _exhaustive;
          }
        }
      }
      default: {
        // Exhaustiveness check - will cause compile error if new context type added
        const _exhaustive: never = context;
        return _exhaustive;
      }
    }
  }, [context]);

  const headerContent = useMemo(
    () => (
      <div className="flex flex-col w-full gap-2">
        {/* Product Switcher - always visible */}
        <div
          className={cn(
            "flex w-full",
            isCollapsed ? "justify-center" : "items-center justify-between gap-4",
          )}
        >
          <ProductSwitcher workspace={props.workspace} currentProduct={currentProduct} />
          {state !== "collapsed" && !isMobile && (
            <button type="button" onClick={toggleSidebar} aria-label="Collapse sidebar">
              <SidebarLeftHide className="text-gray-8" iconSize="xl-medium" />
            </button>
          )}
        </div>

        {/* Resource Heading - only at resource-level, only when expanded */}
        {showResourceHeading && !isCollapsed && context.type === "resource" && (
          <ResourceHeading
            resourceType={context.resourceType}
            resourceId={context.resourceId}
            resourceName={fetchedResourceName || context.resourceName}
          />
        )}
      </div>
    ),
    [
      isCollapsed,
      state,
      isMobile,
      toggleSidebar,
      showResourceHeading,
      context,
      props.workspace,
      currentProduct,
      fetchedResourceName,
    ],
  );

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader className="px-4 items-center pt-4">{headerContent}</SidebarHeader>
      <SidebarContent className="px-2 flex flex-col justify-between">
        {/* Context-aware navigation */}
        <div className="flex flex-col gap-4">
          {state === "collapsed" && (
            <SidebarGroup>
              <SidebarMenu className="gap-2">
                <ToggleSidebarButton toggleNavItem={toggleNavItem} toggleSidebar={toggleSidebar} />

                {/* Back button - only in collapsed state when viewing a resource */}
                {showResourceHeading && context.type === "resource" && (
                  <TooltipProvider>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Link
                          href={`/${workspace.slug}/${RESOURCE_TYPE_ROUTES[context.resourceType]}`}
                          className="flex items-center justify-center h-10 w-10 text-gray-11 hover:text-gray-12 hover:bg-gray-3 rounded-md transition-colors"
                          aria-label={`Back to All ${RESOURCE_TYPE_PLURAL[context.resourceType]}`}
                        >
                          <ChevronLeft className="w-4 h-4" />
                        </Link>
                      </TooltipTrigger>
                      <TooltipContent side="right" sideOffset={8}>
                        <p className="text-xs">
                          Back to All {RESOURCE_TYPE_PLURAL[context.resourceType]}
                        </p>
                      </TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                )}
              </SidebarMenu>
            </SidebarGroup>
          )}

          <ContextNavigation context={context} onResourceNameFetched={handleResourceNameFetched} />

          {/* Workspace section with border-top separator */}
          <div className="border-t border-grayA-4 pt-4">
            <WorkspaceSection />
          </div>
        </div>

        {/* Bottom section: Usage banner */}
        <div className="flex flex-col gap-2">
          <SidebarGroup>
            <UsageBanner quotas={props.workspace.quotas} />
          </SidebarGroup>
        </div>
      </SidebarContent>
      <SidebarFooter>
        {/* Workspace switcher with help button - hidden on mobile */}
        {!isMobile && (
          <div
            className={cn("flex items-center gap-2", isCollapsed ? "flex-col" : "justify-between")}
          >
            <WorkspaceSwitcher />
            <HelpButton />
          </div>
        )}
      </SidebarFooter>
    </Sidebar>
  );
}
