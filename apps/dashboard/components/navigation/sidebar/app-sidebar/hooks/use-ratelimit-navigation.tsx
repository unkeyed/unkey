"use client";

import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { trpc } from "@/lib/trpc/client";
import {
  ArrowDottedRotateAnticlockwise,
  ArrowOppositeDirectionY,
  Gear,
  Layers3,
} from "@unkey/icons";
import { useSelectedLayoutSegments } from "next/navigation";
import { useMemo } from "react";

const DEFAULT_LIMIT = 10;

export const useRatelimitNavigation = (baseNavItems: NavItem[]) => {
  const segments = useSelectedLayoutSegments() ?? [];

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } =
    trpc.ratelimit.namespace.query.useInfiniteQuery(
      {
        limit: DEFAULT_LIMIT,
      },
      {
        getNextPageParam: (lastPage) => lastPage.nextCursor,
      },
    );

  // Convert ratelimit namespaces data to navigation items with sub-items
  const ratelimitNavItems = useMemo(() => {
    if (!data?.pages) {
      return [];
    }

    return data.pages.flatMap((page) =>
      page.namespaceList.map((namespace) => {
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
      }),
    );
  }, [data?.pages, segments]);

  const enhancedNavItems = useMemo(() => {
    const items = [...baseNavItems];
    const ratelimitsItemIndex = items.findIndex((item) => item.href === "/ratelimits");

    if (ratelimitsItemIndex !== -1) {
      const ratelimitsItem = { ...items[ratelimitsItemIndex] };
      ratelimitsItem.items = [...(ratelimitsItem.items || []), ...ratelimitNavItems];

      if (hasNextPage) {
        ratelimitsItem.items?.push({
          icon: () => null,
          href: "#load-more-ratelimits",
          label: "More",
          active: false,
          loadMoreAction: true,
        });
      }

      items[ratelimitsItemIndex] = ratelimitsItem;
    }

    return items;
  }, [baseNavItems, ratelimitNavItems, hasNextPage]);

  const loadMore = () => {
    if (!isFetchingNextPage && hasNextPage) {
      fetchNextPage();
    }
  };

  return {
    enhancedNavItems,
    isLoading,
    loadMore,
  };
};
