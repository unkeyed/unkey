"use client";
import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useMemo } from "react";

export const useApiNavigation = (baseNavItems: NavItem[]) => {
  const workspace = useWorkspaceNavigation();

  const basePath = `/${workspace.slug}`;

  const enhancedNavItems = useMemo(() => {
    const items = [...baseNavItems];
    const apisItemIndex = items.findIndex((item) => item.href === `${basePath}/apis`);

    if (apisItemIndex !== -1) {
      const apisItem = { ...items[apisItemIndex] };
      // Don't add API items as children - just keep the base APIs nav item
      // Individual APIs are accessed via the main content area, not the sidebar
      items[apisItemIndex] = apisItem;
    }

    return items;
  }, [baseNavItems, basePath]);

  return {
    enhancedNavItems,
  };
};
