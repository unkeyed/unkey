"use client";

import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
import {
  ArrowDottedRotateAnticlockwise,
  ArrowOppositeDirectionY,
  Gear,
  Layers3,
} from "@unkey/icons";
import { useSelectedLayoutSegments } from "next/navigation";
import { useMemo } from "react";

export const useRatelimitNavigation = (baseNavItems: NavItem[]) => {
  const segments = useSelectedLayoutSegments() ?? [];
  const workspace = useWorkspaceNavigation();
  const { data } = useLiveQuery((q) =>
    q
      .from({ namespace: collection.ratelimitNamespaces })
      .orderBy(({ namespace }) => namespace.id, "desc"),
  );

  const basePath = `/${workspace.slug}`;
  // Convert ratelimit namespaces data to navigation items with sub-items
  const ratelimitNavItems = useMemo(() => {
    if (data.length === 0) {
      return [];
    }

    return data.map((namespace) => {
      const rIndex = segments.findIndex((s) => s === "ratelimits");
      const currentNamespaceActive = rIndex !== -1 && segments.at(rIndex + 1) === namespace.id;

      // Create sub-items for logs, settings, and overrides
      const subItems: NavItem[] = [
        {
          icon: ArrowOppositeDirectionY,
          href: `${basePath}/ratelimits/${namespace.id}`,
          label: "Requests",
          active: currentNamespaceActive && !segments.at(3),
        },
        {
          icon: Layers3,
          href: `${basePath}/ratelimits/${namespace.id}/logs`,
          label: "Logs",
          active: currentNamespaceActive && segments.at(3) === "logs",
        },
        {
          icon: Gear,
          href: `${basePath}/ratelimits/${namespace.id}/settings`,
          label: "Settings",
          active: currentNamespaceActive && segments.at(3) === "settings",
        },
        {
          icon: ArrowDottedRotateAnticlockwise,
          href: `${basePath}/ratelimits/${namespace.id}/overrides`,
          label: "Overrides",
          active: currentNamespaceActive && segments.at(3) === "overrides",
        },
      ];

      // Create the main namespace nav item
      const namespaceNavItem: NavItem = {
        icon: null,
        href: `${basePath}/ratelimits/${namespace.id}`,
        label: namespace.name,
        active: currentNamespaceActive,
        // Include sub-items
        items: subItems,
      };

      return namespaceNavItem;
    });
  }, [data, segments, basePath]);

  const enhancedNavItems = useMemo(() => {
    const items = [...baseNavItems];
    const ratelimitsItemIndex = items.findIndex((item) => item.href === `${basePath}/ratelimits`);

    if (ratelimitsItemIndex !== -1) {
      const ratelimitsItem = { ...items[ratelimitsItemIndex] };
      ratelimitsItem.items = [...(ratelimitsItem.items || []), ...ratelimitNavItems];

      items[ratelimitsItemIndex] = ratelimitsItem;
    }

    return items;
  }, [baseNavItems, ratelimitNavItems, basePath]);

  return {
    enhancedNavItems,
  };
};
