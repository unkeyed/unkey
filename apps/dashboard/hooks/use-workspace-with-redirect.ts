"use client";

import type { Router } from "@/lib/trpc/routers";
import { useWorkspace } from "@/providers/workspace-provider";
import type { TRPCClientErrorLike } from "@trpc/client";
import type { Quotas, Workspace } from "@unkey/db";
import { redirect } from "next/navigation";
import { useEffect } from "react";

interface UseWorkspaceWithRedirectOptions {
  /**
   * Whether to redirect to /new when workspace is missing or there's an error
   * @default true
   */
  redirectOnMissing?: boolean;
  /**
   * Custom redirect path when workspace is missing
   * @default "/new"
   */
  redirectTo?: string;
}

interface UseWorkspaceWithRedirectReturnWithRedirect {
  workspace: Workspace;
  quotas: Quotas | null;
  isLoading: boolean;
  error: TRPCClientErrorLike<Router> | null;
  refetch: () => void;
  clearCache: () => void;
}

interface UseWorkspaceWithRedirectReturnWithoutRedirect {
  workspace: Workspace | null;
  quotas: Quotas | null;
  isLoading: boolean;
  error: TRPCClientErrorLike<Router> | null;
  refetch: () => void;
  clearCache: () => void;
}

export function useWorkspaceWithRedirect(
  options: UseWorkspaceWithRedirectOptions & { redirectOnMissing: false },
): UseWorkspaceWithRedirectReturnWithoutRedirect;
export function useWorkspaceWithRedirect(
  options?: UseWorkspaceWithRedirectOptions,
): UseWorkspaceWithRedirectReturnWithRedirect;
export function useWorkspaceWithRedirect(
  options: UseWorkspaceWithRedirectOptions = {},
): UseWorkspaceWithRedirectReturnWithRedirect | UseWorkspaceWithRedirectReturnWithoutRedirect {
  const { redirectOnMissing = true, redirectTo = "/new" } = options;

  const { workspace, error, isLoading, quotas, refetch, clearCache } = useWorkspace();

  useEffect(() => {
    if (!redirectOnMissing || isLoading) {
      return;
    }

    if (error || !workspace) {
      redirect(redirectTo);
    }
  }, [workspace, error, isLoading, redirectOnMissing, redirectTo]);

  if (!redirectOnMissing) {
    return {
      workspace,
      quotas,
      isLoading,
      error,
      refetch,
      clearCache,
    };
  }

  return {
    // biome-ignore lint: This will always hae a value, and is needed to make the hook useful.
    workspace: workspace!,
    quotas,
    isLoading,
    error,
    refetch,
    clearCache,
  };
}
