"use client";

import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/data-provider";
import { useNavbarVariant } from "@/hooks/use-navbar-variant";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import { generateFakeDeployments } from "@/lib/fake-deployments";
import { useMemo } from "react";

type Result = {
  deployments: Deployment[];
  isFake: boolean;
  isLoading: boolean;
};

/**
 * Prototype-only. Returns `useProjectData().deployments` unchanged when
 * they exist. When the list is empty **and** v2b is the active variant,
 * falls back to deterministic fakes so the populated 3-panel layout
 * is reachable for projects that haven't deployed anything yet.
 *
 * Any other variant (v2 / v3 / current) sees the real empty array and
 * keeps the existing "No Deployments Found" empty state.
 */
export function useDeploymentsWithFallback(): Result {
  const { projectId, deployments, isDeploymentsLoading } = useProjectData();
  const { variant } = useNavbarVariant();

  return useMemo<Result>(() => {
    if (isDeploymentsLoading) {
      return { deployments, isFake: false, isLoading: true };
    }
    if (deployments.length === 0 && variant === "v2b") {
      return {
        deployments: generateFakeDeployments(projectId),
        isFake: true,
        isLoading: false,
      };
    }
    return { deployments, isFake: false, isLoading: false };
  }, [deployments, isDeploymentsLoading, projectId, variant]);
}
