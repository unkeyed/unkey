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

  // Fetch the specific API by ID
  const { data, isLoading } = trpc.api.queryApiKeyDetails.useQuery(
    {
      apiId: apiId ?? "",
    },
    {
      enabled: !!apiId,
    },
  );

  const enhancedNavItems = useMemo(() => {
    if (!apiId) {
      return baseNavItems;
    }

    // If loading, mark the item as loading
    if (isLoading) {
      return baseNavItems.map((item) => {
        const hasKeysChild = item.items?.some((child) => child.label === "Keys");
        if (hasKeysChild) {
          return {
            ...item,
            loading: true,
          };
        }
        return item;
      });
    }

    if (!data?.currentApi) {
      return baseNavItems;
    }

    const api = data.currentApi;

    // Find and update the parent API item and its Keys child
    return baseNavItems.map((item) => {
      // Check if this is the API parent item by checking if it has children with Keys
      const hasKeysChild = item.items?.some((child) => child.label === "Keys");

      if (hasKeysChild && item.items) {
        // Update the parent label with the actual API name
        const updatedItem = {
          ...item,
          label: api.name,
          loading: false,
        };

        // Update the Keys child item
        updatedItem.items = item.items.map((childItem) => {
          if (childItem.label === "Keys") {
            // If keyAuthId is null, keep the item disabled (API has no keys yet)
            if (!api.keyAuthId) {
              return {
                ...childItem,
                disabled: true,
                href: `/${workspace.slug}/apis/${apiId}`, // Keep placeholder href
              };
            }

            // Update with the actual keyAuthId (keyspaceId)
            return {
              ...childItem,
              href: `/${workspace.slug}/apis/${apiId}/keys/${api.keyAuthId}`,
              disabled: false,
            };
          }
          return childItem;
        });

        return updatedItem;
      }
      return item;
    });
  }, [baseNavItems, data, apiId, workspace.slug, isLoading]);

  // Get the API name if available
  const apiName = useMemo(() => {
    return data?.currentApi?.name;
  }, [data]);

  return {
    enhancedNavItems,
    apiName,
  };
};
