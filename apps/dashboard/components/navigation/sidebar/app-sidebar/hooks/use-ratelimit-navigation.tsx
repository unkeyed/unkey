"use client";

import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
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

  const { data } = useLiveQuery(
    (q) =>
      q
        .from({ namespace: collection.ratelimitNamespaces })
        .orderBy(({ namespace }) => namespace.id, "desc"),
    // Deps are required here otherwise it won't get rerendered
    [collection.ratelimitNamespaces],
  );

  // Convert ratelimit namespaces data to navigation items with sub-items
  const ratelimitNavItems = useMemo(() => {
    if (data.length === 0) {
      return [];
    }

    return data.map((namespace) => {
      const currentNamespaceActive =
        segments.at(0) === "ratelimits" && segments.at(1) === namespace.id;

      const isExactlyRatelimitRoot = currentNamespaceActive && segments.length === 2;

      // Create sub-items for logs, settings, and overrides
      const subItems: NavItem[] = [
        {
          icon: ArrowOppositeDirectionY,
          href: `/ratelimits/${namespace.id}`,
          label: "Requests",
          active: isExactlyRatelimitRoot || (currentNamespaceActive && !segments.at(2)),
        },
        {
          icon: Layers3,
          href: `/ratelimits/${namespace.id}/logs`,
          label: "Logs",
          active: currentNamespaceActive && segments.at(2) === "logs",
        },
        {
          icon: Gear,
          href: `/ratelimits/${namespace.id}/settings`,
          label: "Settings",
          active: currentNamespaceActive && segments.at(2) === "settings",
        },
        {
          icon: ArrowDottedRotateAnticlockwise,
          href: `/ratelimits/${namespace.id}/overrides`,
          label: "Overrides",
          active: currentNamespaceActive && segments.at(2) === "overrides",
        },
      ];

      // Create the main namespace nav item
      const namespaceNavItem: NavItem = {
        icon: null,
        href: `/ratelimits/${namespace.id}`,
        label: namespace.name,
        active: currentNamespaceActive,
        // Include sub-items
        items: subItems,
      };

      return namespaceNavItem;
    });
  }, [data, segments]);

  const enhancedNavItems = useMemo(() => {
    const items = [...baseNavItems];
    const ratelimitsItemIndex = items.findIndex((item) => item.href === "/ratelimits");

    if (ratelimitsItemIndex !== -1) {
      const ratelimitsItem = { ...items[ratelimitsItemIndex] };
      ratelimitsItem.items = [...(ratelimitsItem.items || []), ...ratelimitNavItems];

      items[ratelimitsItemIndex] = ratelimitsItem;
    }

    return items;
  }, [baseNavItems, ratelimitNavItems]);

  return {
    enhancedNavItems,
  };
};
