"use client";

import type { AuthenticatedUser } from "@/lib/auth/types";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import { baseQueryOptions, createRetryFn } from "@/lib/utils/trpc";
import type { TRPCClientErrorLike } from "@trpc/client";
import type { Quotas, Workspace } from "@unkey/db";
import { usePathname } from "next/navigation";
import type React from "react";
import {
  type PropsWithChildren,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
} from "react";

interface WorkspaceContextType {
  user: AuthenticatedUser | null;
  workspace: Workspace | null;
  quotas: Quotas | null;
  isLoading: boolean;
  error: TRPCClientErrorLike<Router> | null;
  refetch: () => void;
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
  const pathname = usePathname();

  // Get user state first
  const userQuery = trpc.user.getCurrentUser.useQuery(undefined, {
    ...baseQueryOptions,
    retry: createRetryFn(2),
    refetchInterval: 1000 * 60 * 10, // 10 minutes
  });

  const { data: user, isLoading: userLoading, error: userError } = userQuery;
  const shouldEnableWorkspaceQuery = useMemo(
    () => Boolean(!userLoading && user?.id && user?.orgId && !userError),
    [userLoading, user?.id, user?.orgId, userError],
  );

  const workspaceQuery = trpc.workspace.getCurrent.useQuery(undefined, {
    ...baseQueryOptions,
    enabled: shouldEnableWorkspaceQuery,
    retry: (failureCount, error) => {
      return createRetryFn(5)(failureCount, error);
    },
  });

  const { data: workspace, isLoading: workspaceLoading, error: workspaceError } = workspaceQuery;

  /**
   *
   * fetches the userQuery on login redirect.
   */
  useEffect(() => {
    const isOnApisRoute = pathname === "/apis";

    if (isOnApisRoute && !userLoading && !user) {
      userQuery.refetch();
    }
  }, [pathname, userLoading, user, userQuery.refetch]);

  const refetch = useCallback(async () => {
    await Promise.all([userQuery.refetch(), workspaceQuery.refetch()]);
  }, [userQuery.refetch, workspaceQuery.refetch]);

  const value: WorkspaceContextType = useMemo(() => {
    const isLoading =
      userLoading || (shouldEnableWorkspaceQuery && workspaceLoading && !workspaceError);

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

  return <WorkspaceContext.Provider value={value}>{children}</WorkspaceContext.Provider>;
};
