"use client";

import type { AuthenticatedUser } from "@/lib/auth/types";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import {
  baseQueryOptions,
  createRetryFn,
  isWorkOSRedirect,
} from "@/lib/utils/trpc";
import { useQueryClient } from "@tanstack/react-query";
import type { TRPCClientErrorLike } from "@trpc/client";
import type { Quotas, Workspace } from "@unkey/db";
import type React from "react";
import {
  type PropsWithChildren,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
} from "react";

interface WorkspaceContextType {
  user: AuthenticatedUser | null;
  workspace: Workspace | null;
  quotas: Quotas | null;
  isLoading: boolean;
  error: TRPCClientErrorLike<Router> | null;
  refetch: () => void;
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
  const queryClient = useQueryClient();
  const previousUserIdRef = useRef<string | null>(null);

  // Get user state first
  const userQuery = trpc.user.getCurrentUser.useQuery(undefined, {
    ...baseQueryOptions,
    retry: createRetryFn(2),
    refetchInterval: 1000 * 60 * 10, // 10 minutes
  });

  const { data: user, isLoading: userLoading, error: userError } = userQuery;

  // Only fetch workspace if user data is ready and valid
  const shouldEnableWorkspaceQuery = useMemo(
    () => Boolean(!userLoading && user?.id && user?.orgId && !userError),
    [userLoading, user?.id, user?.orgId, userError]
  );

  const workspaceQuery = trpc.workspace.getCurrent.useQuery(undefined, {
    ...baseQueryOptions,
    enabled: shouldEnableWorkspaceQuery,
    retry: createRetryFn(3), // Allow one extra retry for workspace
    refetchOnMount: true, // Ensure query runs on mount if conditions are met
    staleTime: 0, // Prevent stale data issues in production
  });

  const {
    data: workspace,
    isLoading: workspaceLoading,
    error: workspaceError,
  } = workspaceQuery;

  // Explicitly trigger workspace query when conditions are met (production fix)
  useEffect(() => {
    if (
      shouldEnableWorkspaceQuery &&
      !workspaceLoading &&
      !workspace &&
      !workspaceError
    ) {
      console.log(
        "[WorkspaceProvider] Manually triggering workspace refetch for production"
      );
      workspaceQuery.refetch();
    }
  }, [
    shouldEnableWorkspaceQuery,
    workspaceLoading,
    workspace,
    workspaceError,
    workspaceQuery.refetch,
  ]);

  // Memoize refetch function to prevent unnecessary re-renders
  const refetch = useCallback(async () => {
    await Promise.all([userQuery.refetch(), workspaceQuery.refetch()]);
  }, [userQuery.refetch, workspaceQuery.refetch]);

  // Monitor user ID changes and clear cache when user switches
  useEffect(() => {
    const currentUserId = user?.id || null;
    const previousUserId = previousUserIdRef.current;

    // If user changed (including from null to user or user to null)
    if (previousUserId !== null && currentUserId !== previousUserId) {
      console.log("[WorkspaceProvider] User ID changed, clearing cache", {
        previousUserId,
        currentUserId,
      });
      queryClient.clear();
    }

    previousUserIdRef.current = currentUserId;
  }, [user?.id, queryClient]);

  // Monitor authentication failures and clear cache when needed
  useEffect(() => {
    const hasAuthError =
      userError?.data?.code === "UNAUTHORIZED" ||
      userError?.data?.code === "FORBIDDEN" ||
      workspaceError?.data?.code === "UNAUTHORIZED" ||
      workspaceError?.data?.code === "FORBIDDEN";

    if (hasAuthError) {
      console.log("[WorkspaceProvider] Auth error detected, clearing cache", {
        userError: userError?.data?.code,
        workspaceError: workspaceError?.data?.code,
      });
      // Clear all queries when auth fails to prevent stale data
      queryClient.clear();
    }
  }, [userError, workspaceError, queryClient]);

  // Compute context value with proper error handling
  const value: WorkspaceContextType = useMemo(() => {
    // Handle WorkOS redirects by showing loading state
    const hasWorkOSRedirect =
      isWorkOSRedirect(userError) || isWorkOSRedirect(workspaceError);

    if (hasWorkOSRedirect) {
      return {
        user: null,
        workspace: null,
        quotas: null,
        isLoading: true,
        error: null,
        refetch,
      };
    }

    const isLoading =
      userLoading ||
      (shouldEnableWorkspaceQuery && (workspaceLoading || !workspace));

    const error = isLoading ? null : userError || workspaceError || null;

    return {
      user: user ?? null,
      workspace: workspace ?? null,
      quotas: workspace?.quotas ?? null,
      isLoading,
      error,
      refetch,
    };
  }, [
    user,
    workspace,
    userLoading,
    workspaceLoading,
    shouldEnableWorkspaceQuery,
    userError,
    workspaceError,
    refetch,
  ]);

  return (
    <WorkspaceContext.Provider value={value}>
      {children}
    </WorkspaceContext.Provider>
  );
};
