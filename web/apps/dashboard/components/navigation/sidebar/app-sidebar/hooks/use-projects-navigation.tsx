"use client";
import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useMemo } from "react";

export const useProjectNavigation = (baseNavItems: NavItem[]) => {
  const workspace = useWorkspaceNavigation();

  const basePath = `/${workspace.slug}/projects`;

  const enhancedNavItems = useMemo(() => {
    const items = [...baseNavItems];
    const projectsItemIndex = items.findIndex((item) => item.href === basePath);

    if (projectsItemIndex !== -1) {
      const projectsItem = { ...items[projectsItemIndex] };
      // Don't add project items as children - just keep the base Projects nav item
      // Individual projects are accessed via the main content area, not the sidebar
      items[projectsItemIndex] = projectsItem;
    }

    return items;
  }, [baseNavItems, basePath]);

  return {
    enhancedNavItems,
  };
};
