"use client";
import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";

/**
 * Hook to enhance API resource navigation with keyspace ID.
 * Fetches the API data to get the keyspaceId and updates the Keys link.
 * If the API doesn't have a keyspace (keyspaceId is null), the Keys link remains disabled.
 */
export const useApiKeyspace = (baseNavItems: NavItem[], apiId?: string) => {
  const workspace = useWorkspaceNavigation();

  // Fetch all APIs to find the one we need
  const { data } = trpc.api.overview.query.useInfiniteQuery(
    {
      limit: 18, // Max allowed by the API
    },
    {
      enabled: !!apiId,
      getNextPageParam: (lastPage) => lastPage.nextCursor,
    },
  );

  const enhancedNavItems = useMemo(() => {
    if (!data?.pages || !apiId) {
      return baseNavItems;
    }

    // Find the API in the fetched data
    const api = data.pages.flatMap((page) => page.apiList).find((a) => a.id === apiId);

    // If API not found, return base items unchanged
    if (!api) {
      return baseNavItems;
    }

    // Find and update the parent API item and its Keys child
    return baseNavItems.map((item) => {
      // Check if this is the API parent item by checking if it has children with Keys
      const hasKeysChild = item.items?.some((child) => child.label === "Keys");

      if (hasKeysChild && item.items) {
        // Update the parent label with the actual API name
        const updatedItem = {
          ...item,
          label: api.name,
        };

        // Update the Keys child item
        updatedItem.items = item.items.map((childItem) => {
          if (childItem.label === "Keys") {
            // If keyspaceId is null, keep the item disabled (API has no keys yet)
            if (!api.keyspaceId) {
              return {
                ...childItem,
                disabled: true,
                href: `/${workspace.slug}/apis/${apiId}`, // Keep placeholder href
              };
            }

            // Update with the actual keyspaceId
            return {
              ...childItem,
              href: `/${workspace.slug}/apis/${apiId}/keys/${api.keyspaceId}`,
              disabled: false,
            };
          }
          return childItem;
        });

        return updatedItem;
      }
      return item;
    });
  }, [baseNavItems, data, apiId, workspace.slug]);

  // Get the API name if available
  const apiName = useMemo(() => {
    if (!data?.pages || !apiId) {
      return undefined;
    }
    const api = data.pages.flatMap((page) => page.apiList).find((a) => a.id === apiId);
    return api?.name;
  }, [data, apiId]);

  return {
    enhancedNavItems,
    apiName,
  };
};
