"use client";

import { trpc } from "@/lib/trpc/client";
import type { Quotas, Workspace } from "@unkey/db";
import type React from "react";
import { type PropsWithChildren, createContext, useCallback, useContext, useMemo } from "react";

interface WorkspaceContextType {
  workspace: Workspace | null;
  quotas: Quotas | null;
  isLoading: boolean;
  error: Error | null;
  refetch: () => void;
  clearCache: () => void;
}

const WorkspaceContext = createContext<WorkspaceContextType | undefined>(undefined);

export const useWorkspace = () => {
  const context = useContext(WorkspaceContext);
  if (context === undefined) {
    throw new Error("useWorkspace must be used within a WorkspaceProvider");
  }
  return context;
};

export const WorkspaceProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const {
    data: workspace,
    isLoading,
    error,
    refetch: trpcRefetch,
    isFetching,
  } = trpc.workspace.getCurrent.useQuery(undefined, {
    staleTime: 1000 * 60 * 5, // 5 minutes
    cacheTime: 1000 * 60 * 15, // 15 minutes
    retry: 2,
    refetchOnWindowFocus: false,
    refetchOnReconnect: true,
  });

  const clearCache = useCallback(() => {
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
    [workspace, isLoading, isFetching, error, refetch, clearCache],
  );

  return <WorkspaceContext.Provider value={value}>{children}</WorkspaceContext.Provider>;
};
