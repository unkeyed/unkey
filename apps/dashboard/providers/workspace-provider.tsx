"use client";

import { trpc } from "@/lib/trpc/client";
import type { Quotas, Workspace } from "@unkey/db";
import type React from "react";
import {
  type PropsWithChildren,
  createContext,
  useCallback,
  useContext,
  useMemo,
  useEffect,
} from "react";

interface WorkspaceContextType {
  workspace: Workspace | null;
  quotas: Quotas | null;
  isLoading: boolean;
  error: Error | null;
  refetch: () => void;
  clearCache: () => void;
}

const WorkspaceContext = createContext<WorkspaceContextType | undefined>(
  undefined
);

export const useWorkspace = () => {
  const context = useContext(WorkspaceContext);
  if (context === undefined) {
    throw new Error("useWorkspace must be used within a WorkspaceProvider");
  }
  return context;
};

export const WorkspaceProvider: React.FC<PropsWithChildren> = ({
  children,
}) => {
  const {
    data: workspace,
    isLoading,
    error,
    refetch: trpcRefetch,
    isFetching,
    dataUpdatedAt,
    isStale,
    isPreviousData,
  } = trpc.workspace.getCurrent.useQuery(undefined, {
    staleTime: 1000 * 60 * 5, // 5 minutes
    cacheTime: 1000 * 60 * 15, // 15 minutes
    retry: (failureCount, error) => {
      if (
        error?.message?.includes("workspace not found in context") ||
        error?.data?.code === "NOT_FOUND" ||
        error?.data?.code === "UNAUTHORIZED"
      ) {
        return false;
      }
      return failureCount < 2;
    },
    refetchOnWindowFocus: false,
    refetchOnReconnect: true,
    // Cache debugging
    onSuccess: (data) => {
      const now = new Date().toISOString();
      const cacheAge = dataUpdatedAt
        ? Math.round((Date.now() - dataUpdatedAt) / 1000)
        : 0;
      console.log(`🎯 [${now}] Workspace query SUCCESS:`, {
        workspaceName: data?.name || "no-workspace",
        isFetching,
        isFromCache: !isFetching && cacheAge > 1, // If not fetching and data is older than 1s, likely from cache
        cacheAgeSeconds: cacheAge,
        dataTimestamp: dataUpdatedAt
          ? new Date(dataUpdatedAt).toISOString()
          : "never",
      });
    },
    onError: (error) => {
      console.log(`❌ [${new Date().toISOString()}] Workspace query ERROR:`, {
        message: error.message,
        code: error?.data?.code,
        isFetching,
      });

      if (
        error?.message?.includes("workspace not found in context") ||
        error?.data?.code === "NOT_FOUND"
      ) {
        // These are expected during initial load - don't log as errors
        console.debug(
          "Workspace not available in context - user may need to create workspace"
        );
      }
    },
  });

  // Add cache monitoring effect
  useEffect(() => {
    const now = new Date().toISOString();
    const cacheAge = dataUpdatedAt
      ? Math.round((Date.now() - dataUpdatedAt) / 1000)
      : 0;
    const staleThreshold = 5 * 60; // 5 minutes in seconds

    console.log(`📊 [${now}] Workspace cache status:`, {
      isLoading,
      isFetching,
      isStale,
      isPreviousData,
      hasData: !!workspace,
      workspaceName: workspace?.name || "no-workspace",
      cacheAgeSeconds: cacheAge,
      isWithinStaleTime: cacheAge < staleThreshold,
      dataTimestamp: dataUpdatedAt
        ? new Date(dataUpdatedAt).toISOString()
        : "never",
      cacheStatus: (() => {
        if (isLoading && !workspace) return "INITIAL_LOADING";
        if (isFetching && workspace) return "REFETCHING";
        if (!isFetching && workspace && cacheAge < staleThreshold)
          return "SERVING_FROM_CACHE";
        if (!isFetching && workspace && cacheAge >= staleThreshold)
          return "STALE_DATA";
        if (isPreviousData) return "SHOWING_PREVIOUS_DATA";
        return "UNKNOWN";
      })(),
    });
  }, [
    isLoading,
    isFetching,
    isStale,
    isPreviousData,
    workspace,
    dataUpdatedAt,
  ]);

  const clearCache = useCallback(() => {
    console.log(`🗑️ [${new Date().toISOString()}] Clearing workspace cache`);

    // Clear any client-side workspace caches
    try {
      Object.keys(localStorage)
        .filter((key) => key.includes("workspace_cache"))
        .forEach((key) => localStorage.removeItem(key));
    } catch (_error) {
      // don't throw error
    }

    // Force refetch from server
    return trpcRefetch();
  }, [trpcRefetch]);

  const refetch = useCallback(async () => {
    console.log(
      `🔄 [${new Date().toISOString()}] Manual workspace refetch triggered`
    );
    return trpcRefetch();
  }, [trpcRefetch]);

  const value: WorkspaceContextType = useMemo(
    () => ({
      workspace: workspace ?? null,
      quotas: workspace?.quotas ?? null,
      isLoading: isLoading || isFetching,
      error: error as Error | null,
      refetch,
      clearCache,
    }),
    [workspace, isLoading, isFetching, error, refetch, clearCache]
  );

  return (
    <WorkspaceContext.Provider value={value}>
      {children}
    </WorkspaceContext.Provider>
  );
};
