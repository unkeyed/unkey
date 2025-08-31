import type { Identity } from "@/lib/collections/identities";
import { createIdentitiesCollection, searchIdentities } from "@/lib/collections/identities";
import { trpc } from "@/lib/trpc/client";
import { useLiveQuery } from "@tanstack/react-db";
import { useEffect, useMemo, useState } from "react";

type UseIdentitiesParams = {
  workspaceId: string;
  search?: string;
  limit?: number;
  enabled?: boolean;
};

export function useIdentities({
  workspaceId,
  search,
  limit = 50,
  enabled = true,
}: UseIdentitiesParams) {
  const [isSearching, setIsSearching] = useState(false);
  const [searchResults, setSearchResults] = useState<Identity[]>([]);

  // Determine if we should use search or regular query
  const shouldSearch = search && search.trim().length > 0;

  // Fetch regular identities list using tRPC
  const {
    data: tRPCData,
    isLoading: tRPCLoading,
    isError: tRPCError,
  } = trpc.identity.query.useQuery(
    {
      limit,
    },
    {
      enabled: enabled && !shouldSearch && !!workspaceId,
    },
  );

  // Create the collection for this workspace
  const collection = useMemo(() => {
    if (!enabled || !workspaceId) return null;

    // Initialize with tRPC data if available and not searching
    const initialData = !shouldSearch && tRPCData?.identities ? tRPCData.identities : [];
    return createIdentitiesCollection(workspaceId, initialData);
  }, [workspaceId, enabled, tRPCData?.identities, shouldSearch]);

  // Handle search separately
  useEffect(() => {
    if (!shouldSearch || !search) {
      setSearchResults([]);
      setIsSearching(false);
      return;
    }

    let cancelled = false;
    setIsSearching(true);

    const performSearch = async () => {
      try {
        const results = await searchIdentities(search.trim());
        if (!cancelled) {
          setSearchResults(results);

          // Update collection with search results
          if (collection) {
            // Clear existing items
            const existingItems = collection.toArray;
            existingItems.forEach((item) => {
              collection.delete(item.id);
            });

            // Insert search results
            results.forEach((identity) => {
              collection.insert(identity);
            });
          }
        }
      } catch (error) {
        console.error("Search failed:", error);
        if (!cancelled) {
          setSearchResults([]);
        }
      } finally {
        if (!cancelled) {
          setIsSearching(false);
        }
      }
    };

    // Debounce search
    const timeoutId = setTimeout(performSearch, 300);

    return () => {
      cancelled = true;
      clearTimeout(timeoutId);
    };
  }, [search, shouldSearch, collection]);

  // Update collection when tRPC data changes (for non-search cases)
  useEffect(() => {
    if (collection && tRPCData?.identities && !shouldSearch) {
      // Clear existing data and insert fresh data from tRPC
      const currentItems = collection.toArray;
      const currentKeys = currentItems.map((item) => item.id);

      // Remove items that are no longer in the tRPC response
      currentKeys.forEach((key) => {
        if (!tRPCData.identities.find((item) => item.id === key)) {
          collection.delete(key);
        }
      });

      // Insert or update items from tRPC response
      tRPCData.identities.forEach((identity) => {
        const existing = currentItems.find((item) => item.id === identity.id);
        if (existing) {
          // Update existing identity
          collection.update(identity.id, () => identity);
        } else {
          // Insert new identity
          collection.insert(identity);
        }
      });
    }
  }, [collection, tRPCData?.identities, shouldSearch]);

  // Use live query to get reactive data from the collection
  const liveQueryResult = collection
    ? useLiveQuery(collection)
    : { data: [], isLoading: false, isError: false };

  const {
    data: identities,
    isError: liveQueryError,
    isLoading: liveQueryLoading,
  } = liveQueryResult;

  // Transform the data to match the expected format
  const transformedIdentities = useMemo(() => {
    if (shouldSearch) {
      return searchResults;
    }

    if (!identities || !Array.isArray(identities)) return [];
    return identities as Identity[];
  }, [identities, searchResults, shouldSearch]);

  return {
    identities: transformedIdentities,
    isLoading: shouldSearch ? isSearching : tRPCLoading || liveQueryLoading,
    error: tRPCError || liveQueryError,
    hasMore: shouldSearch ? false : (tRPCData?.hasMore ?? false),
    nextCursor: shouldSearch ? undefined : tRPCData?.nextCursor,
    // Helper methods for optimistic mutations
    createIdentity: (identity: Identity) => {
      if (!collection) return;
      collection.insert(identity);
    },
    updateIdentity: (identityId: string, updates: Partial<Identity>) => {
      if (!collection) return;
      collection.update(identityId, (draft) => {
        Object.assign(draft, updates);
      });
    },
    deleteIdentity: (identityId: string) => {
      if (!collection) return;
      collection.delete(identityId);
    },
  };
}
