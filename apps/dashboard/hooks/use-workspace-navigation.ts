"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

/**
 * Custom hook that handles navigation logic based on workspace and user state
 * This centralizes redirect logic and can be reused across components
 */
export const useWorkspaceNavigation = () => {
  const router = useRouter();
  const { user, workspace, isLoading, error } = useWorkspace();

  useEffect(() => {
    // Don't navigate while loading
    if (isLoading) {
      return;
    }

    // Handle authentication errors
    const isAuthError = error?.data?.code === "UNAUTHORIZED" || error?.data?.code === "FORBIDDEN";

    if (isAuthError) {
      router.push("/auth/sign-in");
      return;
    }

    // Handle cases where user needs workspace setup
    if (user && (!user.orgId || !user.role || (!workspace && !error))) {
      router.push("/new");
      return;
    }
  }, [user, workspace, isLoading, error, router]);

  return {
    isReady: !isLoading && workspace && user,
    needsAuth: error?.data?.code === "UNAUTHORIZED" || error?.data?.code === "FORBIDDEN",
    needsWorkspaceSetup: user && (!user.orgId || !user.role || (!workspace && !error)),
  };
};
