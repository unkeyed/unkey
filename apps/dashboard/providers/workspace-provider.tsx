"use client";

import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import type { TRPCClientErrorLike } from "@trpc/client";
import type { Quotas, Workspace } from "@unkey/db";
import type React from "react";
import {
  type PropsWithChildren,
  createContext,
  useCallback,
  useContext,
  useMemo,
  useRef,
} from "react";

interface WorkspaceContextType {
  workspace: Workspace | null;
  quotas: Quotas | null;
  isLoading: boolean;
  error: TRPCClientErrorLike<Router> | null;
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
  const retryCountRef = useRef(0);
  const lastErrorRef = useRef<TRPCClientErrorLike<Router> | null>(null);

  const {
    data: workspace,
    isLoading,
    error,
    refetch: trpcRefetch,
    isFetching,
  } = trpc.workspace.getCurrent.useQuery(undefined, {
    staleTime: 1000 * 60 * 5, // 5 minutes
    cacheTime: 1000 * 60 * 15, // 15 minutes
    retry: (failureCount, error) => {
      // Don't retry on definitive auth/workspace errors
      if (error?.data?.code === "UNAUTHORIZED" || error?.data?.code === "FORBIDDEN") {
        return false;
      }

      // For "workspace not found in context" errors, retry up to 3 times
      // This handles first-time login race conditions
      if (error?.message?.includes("workspace not found in context")) {
        return failureCount < 3;
      }

      // For NOT_FOUND errors, retry once in case it's a timing issue
      if (error?.data?.code === "NOT_FOUND") {
        return failureCount < 1;
      }

      // For other errors, retry twice
      return failureCount < 2;
    },
    retryDelay: (attemptIndex) => {
      // Exponential backoff: 500ms, 1s, 2s
      return Math.min(500 * 2 ** attemptIndex, 2000);
    },
    refetchOnWindowFocus: false,
    refetchOnReconnect: true,
    // Return null instead of throwing on certain errors
    onError: (error) => {
      lastErrorRef.current = error;
      if (
        error?.message?.includes("workspace not found in context") ||
        error?.data?.code === "NOT_FOUND"
      ) {
        // These might be expected during initial load - log as debug
        console.debug(
          "Workspace not available in context - user may need to create workspace",
          error,
        );
      } else {
        // Log other errors for debugging
        console.warn("Workspace loading error:", error);
      }
    },
    onSuccess: () => {
      // Reset retry count and error on successful fetch
      retryCountRef.current = 0;
      lastErrorRef.current = null;
    },
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

    // Reset retry tracking
    retryCountRef.current = 0;
    lastErrorRef.current = null;

    // Force refetch from server
    return trpcRefetch();
  }, [trpcRefetch]);

  const refetch = useCallback(async () => {
    // Reset retry tracking on manual refetch
    retryCountRef.current = 0;
    lastErrorRef.current = null;
    return trpcRefetch();
  }, [trpcRefetch]);

  const value: WorkspaceContextType = useMemo(() => {
    // If we're currently loading or fetching, show loading state
    const isCurrentlyLoading = isLoading || isFetching;

    // Enhanced error handling for better user experience
    let processedError: TRPCClientErrorLike<Router> | null = null;
    if (error && !isCurrentlyLoading) {
      // Only surface errors that aren't being automatically retried
      const isRetriableContextError = error?.message?.includes("workspace not found in context");
      const isRetriableNotFound = error?.data?.code === "NOT_FOUND";

      // Don't show error if we're in the middle of automatic retries for these cases
      if (!(isRetriableContextError || isRetriableNotFound) || retryCountRef.current >= 3) {
        processedError = error;
      }
    }

    return {
      workspace: workspace ?? null,
      quotas: workspace?.quotas ?? null,
      isLoading: isCurrentlyLoading,
      error: processedError,
      refetch,
      clearCache,
    };
  }, [workspace, isLoading, isFetching, error, refetch, clearCache]);

  return <WorkspaceContext.Provider value={value}>{children}</WorkspaceContext.Provider>;
};
