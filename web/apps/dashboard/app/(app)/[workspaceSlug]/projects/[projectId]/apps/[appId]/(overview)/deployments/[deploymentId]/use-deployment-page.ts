import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import { useFlag } from "@/lib/flags/provider";
import { routes } from "@/lib/navigation/routes";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";
import { useProjectData } from "../../data-provider";
import { useAppCurrentDeployment } from "../../hooks/use-app-current-deployment";
import { deploymentPageModel } from "./deployment-page-model";
import { useDeployment } from "./layout-provider";
import { useDeploymentStatus } from "./use-deployment-status";

const SUCCESS_BANNER_MS = 4_500;

export type DeploymentPage = ReturnType<typeof useDeploymentPage>;

export function useDeploymentPage() {
  const buildRequested = Boolean(useSearchParams().get("build"));
  const [buildRequestedOnPage, setBuildRequestedOnPage] = useState(buildRequested);
  if (buildRequested && !buildRequestedOnPage) {
    setBuildRequestedOnPage(true);
  }
  const { deployment } = useDeployment();
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const clearRequestedStage = useCallback(
    () =>
      router.replace(
        routes.projects.apps.deployment({
          workspaceSlug: workspace.slug,
          projectId: deployment.projectId,
          appId: deployment.appId,
          deploymentId: deployment.id,
        }),
        { scroll: false },
      ),
    [router, workspace.slug, deployment.projectId, deployment.appId, deployment.id],
  );
  const {
    getDomainsForDeployment,
    customDomains,
    isDomainsLoading,
    isCustomDomainsLoading,
    refetchDomains,
  } = useProjectData();
  const { app } = useAppCurrentDeployment();
  const { steps, derivedStatus } = useDeploymentStatus(deployment);
  const showSuccessBanner = useSuccessBanner(deployment.status);

  const ready = deployment.status === "ready";
  useEffect(() => {
    if (ready) {
      steps.refetch();
      refetchDomains();
    }
  }, [ready, refetchDomains, steps.refetch]);

  const domainsLoading = isDomainsLoading || isCustomDomainsLoading;
  return {
    deployment,
    steps: steps.data,
    domainsLoading,
    showSuccessBanner,
    clearRequestedStage,
    ...deploymentPageModel({
      deployment,
      status: derivedStatus,
      buildRequested,
      buildRequestedOnPage,
      domains: getDomainsForDeployment(deployment.id),
      domainsLoading,
      customDomains,
      currentDeploymentId: app?.currentDeploymentId ?? null,
    }),
  };
}

function useSuccessBanner(status: DeploymentStatus): boolean {
  const showDarksoulsSuccessBanner = useFlag("showDarksoulsSuccessBanner");
  const previousStatus = useRef(status);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    const becameReady =
      showDarksoulsSuccessBanner && previousStatus.current !== "ready" && status === "ready";
    previousStatus.current = status;
    if (!becameReady) {
      return;
    }
    setVisible(true);
    const timer = setTimeout(() => setVisible(false), SUCCESS_BANNER_MS);
    return () => clearTimeout(timer);
  }, [status, showDarksoulsSuccessBanner]);

  return visible;
}
