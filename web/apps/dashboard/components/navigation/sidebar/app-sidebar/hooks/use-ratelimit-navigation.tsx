"use client";

import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useMemo } from "react";

export const useRatelimitNavigation = (baseNavItems: NavItem[]) => {
  const workspace = useWorkspaceNavigation();

  const basePath = `/${workspace.slug}`;

  const enhancedNavItems = useMemo(() => {
    const items = [...baseNavItems];
    const ratelimitsItemIndex = items.findIndex((item) => item.href === `${basePath}/ratelimits`);

    if (ratelimitsItemIndex !== -1) {
      const ratelimitsItem = { ...items[ratelimitsItemIndex] };
      // Don't add namespace items as children - just keep the base Ratelimit nav item
      // Individual namespaces are accessed via the main content area, not the sidebar
      items[ratelimitsItemIndex] = ratelimitsItem;
    }

    return items;
  }, [baseNavItems, basePath]);

  return {
    enhancedNavItems,
  };
};
