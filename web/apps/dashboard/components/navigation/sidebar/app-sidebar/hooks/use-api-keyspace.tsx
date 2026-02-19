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

    // Find and update the Keys nav item
    return baseNavItems.map((item) => {
      if (item.label === "Keys") {
        // If keyspaceId is null, keep the item disabled (API has no keys yet)
        if (!api.keyspaceId) {
          return {
            ...item,
            disabled: true,
            href: `/${workspace.slug}/apis/${apiId}`, // Keep placeholder href
          };
        }

        // Update with the actual keyspaceId
        return {
          ...item,
          href: `/${workspace.slug}/apis/${apiId}/keys/${api.keyspaceId}`,
          disabled: false,
        };
      }
      return item;
    });
  }, [baseNavItems, data?.pages, apiId, workspace.slug]);

  // Get the API name if available
  const apiName = useMemo(() => {
    if (!data?.pages || !apiId) {
      return undefined;
    }
    const api = data.pages.flatMap((page) => page.apiList).find((a) => a.id === apiId);
    return api?.name;
  }, [data?.pages, apiId]);

  return {
    enhancedNavItems,
    apiName,
  };
};
