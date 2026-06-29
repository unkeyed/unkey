"use client";

import type { AuthenticatedUser } from "@/lib/auth/types";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import { baseQueryOptions, createRetryFn, isAuthError } from "@/lib/utils/trpc";
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
  /**
   * True when the server definitively reported that no workspace exists for
   * this session (fresh sign-up, onboarding not finished). Distinct from
   * `error`, which is reserved for failed lookups.
   */
  workspaceMissing: boolean;
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

  // This provider sits in the root layout, so it also wraps the auth pages.
  // Visitors there are signed out by definition: querying the current user
  // would only produce guaranteed 401s, so don't fire it at all.
  const isAuthRoute = pathname?.startsWith("/auth") ?? false;

  // Get user state first
  const userQuery = trpc.user.getCurrentUser.useQuery(undefined, {
    ...baseQueryOptions,
    enabled: !isAuthRoute,
    retry: createRetryFn(2),
    refetchInterval: 1000 * 60 * 10, // 10 minutes
  });

  const { data: user, isLoading: userLoading, error: userError } = userQuery;

  // The server resolves the workspace from the session, not from the user
  // query's result, so this can run in parallel with the user query instead
  // of waterfalling behind it.
  const workspaceQuery = trpc.workspace.getCurrent.useQuery(undefined, {
    ...baseQueryOptions,
    enabled: !isAuthRoute,
    // NOT_FOUND is a definitive answer (no workspace yet), not a transient
    // failure, so retrying it only delays onboarding redirects.
    retry: (failureCount, error) =>
      !isAuthError(error) && error.data?.code !== "NOT_FOUND" && failureCount < 2,
  });

  const { data: workspace, isLoading: workspaceLoading, error: workspaceError } = workspaceQuery;

  // "No workspace" (fresh sign-up, onboarding not finished) is an expected
  // state, distinct from a failed lookup.
  const workspaceMissing = workspaceError?.data?.code === "NOT_FOUND";

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
    // A disabled query reports isLoading=true forever, so ignore loading
    // states on auth routes where the queries never run.
    const isLoading = !isAuthRoute && (userLoading || (workspaceLoading && !workspaceError));

    const error = isLoading ? null : userError || (workspaceMissing ? null : workspaceError);

    return {
      user: user ?? null,
      workspace: workspace ?? null,
      quotas: workspace?.quotas ?? null,
      isLoading,
      error: error ?? null,
      workspaceMissing: !isLoading && workspaceMissing,
      refetch,
    };
  }, [
    user,
    workspace,
    isAuthRoute,
    userLoading,
    workspaceLoading,
    workspaceMissing,
    userError,
    workspaceError,
    refetch,
  ]);

  return <WorkspaceContext.Provider value={value}>{children}</WorkspaceContext.Provider>;
};
