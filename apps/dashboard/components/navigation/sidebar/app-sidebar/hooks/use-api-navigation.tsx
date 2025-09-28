"use client";
import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { ArrowOppositeDirectionY, Gear } from "@unkey/icons";
import { Key } from "lucide-react";
import { useSelectedLayoutSegments } from "next/navigation";
import { useMemo } from "react";

const DEFAULT_LIMIT = 10;

export const useApiNavigation = (baseNavItems: NavItem[]) => {
  const workspace = useWorkspaceNavigation();
  const segments = useSelectedLayoutSegments() ?? [];

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } =
    trpc.api.overview.query.useInfiniteQuery(
      {
        limit: DEFAULT_LIMIT,
      },
      {
        getNextPageParam: (lastPage) => lastPage.nextCursor,
      },
    );

  // Convert API data to navigation items with sub-items for settings and keys
  const apiNavItems = useMemo(() => {
    if (!data?.pages) {
      return [];
    }

    return data.pages.flatMap((page) =>
      page.apiList.map((api) => {
        const aIndex = segments.findIndex((s) => s === "apis");
        const currentApiActive = aIndex !== -1 && segments.at(aIndex + 1) === api.id;

        const settingsItem: NavItem = {
          icon: Gear,
          href: `/${workspace.slug}/apis/${api.id}/settings`,
          label: "Settings",
          active: currentApiActive && segments.at(3) === "settings",
        };

        const overviewItem: NavItem = {
          icon: ArrowOppositeDirectionY,
          href: `/${workspace.slug}/apis/${api.id}`,
          label: "Requests",
          active: currentApiActive && !segments.at(3),
        };

        const subItems: NavItem[] = [overviewItem];

        if (api.keyspaceId) {
          const keysItem: NavItem = {
            icon: Key,
            href: `/${workspace.slug}/apis/${api.id}/keys/${api.keyspaceId}`,
            label: "Keys",
            active: currentApiActive && segments.at(3) === "keys",
          };

          subItems.push(keysItem);
        }

        subItems.push(settingsItem);

        // Create the main API nav item with proper icon setup
        const apiNavItem: NavItem = {
          // This is critical - must provide some icon to ensure chevron renders
          icon: null,
          href: `/${workspace.slug}/apis/${api.id}`,
          label: api.name,
          active: currentApiActive,
          // Always set showSubItems to true to ensure chevron appears
          showSubItems: true,
          // Include sub-items
          items: subItems,
        };

        return apiNavItem;
      }),
    );
  }, [data?.pages, segments, workspace.slug]);

  const enhancedNavItems = useMemo(() => {
    const items = [...baseNavItems];
    const apisItemIndex = items.findIndex((item) => item.href === `/${workspace.slug}/apis`);

    if (apisItemIndex !== -1) {
      const apisItem = { ...items[apisItemIndex] };
      // Always set showSubItems to true for the APIs section
      apisItem.showSubItems = true;
      apisItem.items = [...(apisItem.items || []), ...apiNavItems];

      if (hasNextPage) {
        apisItem.items?.push({
          icon: () => null,
          href: "#load-more-apis",
          label: <div className="font-normal decoration-dotted underline ">More</div>,
          active: false,
          loadMoreAction: true,
        });
      }

      items[apisItemIndex] = apisItem;
    }

    return items;
  }, [baseNavItems, apiNavItems, hasNextPage, workspace.slug]);

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
