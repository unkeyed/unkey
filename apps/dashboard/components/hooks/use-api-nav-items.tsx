"use client";
import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { trpc } from "@/lib/trpc/client";
import { useSelectedLayoutSegments } from "next/navigation";
import { useMemo } from "react";

// Default limit for fetching API items
const DEFAULT_LIMIT = 10;

// Enhanced hook that accepts base navigation items and returns fully constructed navigation
export const useWorkspaceNavigation = (baseNavItems: NavItem[]) => {
  const segments = useSelectedLayoutSegments() ?? [];

  // Use infinite query for pagination
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } =
    trpc.api.overview.query.useInfiniteQuery(
      {
        limit: DEFAULT_LIMIT,
      },
      {
        getNextPageParam: (lastPage) => lastPage.nextCursor,
      },
    );

  // Convert API data to navigation items
  const apiNavItems = useMemo(() => {
    if (!data?.pages) {
      return [];
    }

    return data.pages.flatMap((page) =>
      page.apiList.map((api) => ({
        icon: null,
        href: `/apis/${api.id}`,
        label: api.name,
        active: segments.at(0) === "apis" && segments.at(1) === api.id,
      })),
    );
  }, [data?.pages, segments]);

  const enhancedNavItems = useMemo(() => {
    const items = [...baseNavItems];

    const apisItemIndex = items.findIndex((item) => item.href === "/apis");

    if (apisItemIndex !== -1) {
      const apisItem = { ...items[apisItemIndex] };

      apisItem.items = [...(apisItem.items || []), ...apiNavItems];

      if (hasNextPage) {
        apisItem.items?.push({
          icon: null,
          href: "#load-more",
          label: <div className="font-normal">more</div>,
          active: false,
          loadMoreAction: true,
        });
      }

      items[apisItemIndex] = apisItem;
    }

    return items;
  }, [baseNavItems, apiNavItems, hasNextPage]);

  // Function to handle loading more items
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
