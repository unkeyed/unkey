import { useLiveQuery } from "@tanstack/react-db";
import { trpc } from "@/lib/trpc/client";
import { useEffect, useMemo, useRef } from "react";
import { createApisCollection, type ApiOverviewWithId } from "@/lib/collections/apis";
import type { ApiOverview } from "@/lib/trpc/routers/api/overview/query-overview/schemas";

interface UseApisParams {
  limit?: number;
  enabled?: boolean;
}

interface UseApisReturn {
  data: ApiOverviewWithId[];
  isLoading: boolean;
  error: any;
  total: number;
  hasNextPage: boolean;
  fetchNextPage: () => void;
  isFetchingNextPage: boolean;
  optimisticAdd: (api: ApiOverview) => void;
  optimisticUpdate: (id: string, updates: Partial<ApiOverview>) => void;
  optimisticRemove: (id: string) => void;
}

export function useApis({ limit = 10, enabled = true }: UseApisParams = {}): UseApisReturn {
  const collectionRef = useRef<ReturnType<typeof createApisCollection> | null>(null);

  // Existing tRPC infinite query
  const {
    data: tRPCData,
    isLoading: tRPCLoading,
    error: tRPCError,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = trpc.api.overview.query.useInfiniteQuery(
    { limit },
    {
      enabled,
      getNextPageParam: (lastPage) => lastPage.nextCursor,
    },
  );

  // Flatten tRPC pages data
  const allApis = useMemo(() => {
    if (!tRPCData?.pages) {
      return [];
    }
    return tRPCData.pages.flatMap((page) => page.apiList);
  }, [tRPCData]);

  // Initialize collection when tRPC data is available
  useEffect(() => {
    if (allApis.length > 0 && !collectionRef.current) {
      collectionRef.current = createApisCollection(allApis);
    }
  }, [allApis]);

  // Update collection when tRPC data changes
  useEffect(() => {
    if (collectionRef.current && allApis.length > 0) {
      const collection = collectionRef.current;
      const currentItems = collection.toArray;
      const currentKeys = new Set(currentItems.map((item) => item.id));

      // Remove items that are no longer in tRPC data
      currentKeys.forEach((key) => {
        if (!allApis.find((item) => item.id === key)) {
          collection.delete(key);
        }
      });

      // Insert or update items from tRPC response
      allApis.forEach((api) => {
        const existing = currentItems.find((item) => item.id === api.id);
        const apiWithId = { ...api, id: api.id };
        if (existing) {
          // Update existing API
          collection.update(api.id, () => apiWithId);
        } else {
          // Insert new API
          collection.insert(apiWithId);
        }
      });
    }
  }, [allApis]);

  // Get reactive data from TanStack DB
  const liveQueryResult = collectionRef.current ? 
    useLiveQuery(collectionRef.current) : 
    { data: [], isLoading: false };

  // Use TanStack DB data if available, fallback to tRPC data
  const finalData = liveQueryResult.data.length > 0 ? liveQueryResult.data : 
    allApis.map(api => ({ ...api, id: api.id }));

  const total = tRPCData?.pages[0]?.total || 0;

  // Optimistic mutation functions
  const optimisticAdd = (api: ApiOverview) => {
    if (collectionRef.current) {
      const apiWithId = { ...api, id: api.id };
      collectionRef.current.insert(apiWithId);
    }
  };

  const optimisticUpdate = (id: string, updates: Partial<ApiOverview>) => {
    if (collectionRef.current) {
      const existing = collectionRef.current.toArray.find(item => item.id === id);
      if (existing) {
        const updated = { ...existing, ...updates };
        collectionRef.current.update(id, () => updated);
      }
    }
  };

  const optimisticRemove = (id: string) => {
    if (collectionRef.current) {
      collectionRef.current.delete(id);
    }
  };

  return {
    data: finalData,
    isLoading: tRPCLoading || liveQueryResult.isLoading,
    error: tRPCError,
    total,
    hasNextPage: hasNextPage || false,
    fetchNextPage,
    isFetchingNextPage,
    optimisticAdd,
    optimisticUpdate,
    optimisticRemove,
  };
}