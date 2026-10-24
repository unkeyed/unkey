import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import { useFlag } from "@/lib/flags/provider";
import { useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useProjectData } from "../../data-provider";
import { useAppCurrentDeployment } from "../../hooks/use-app-current-deployment";
import { deploymentPageModel } from "./deployment-page-model";
import { useDeployment } from "./layout-provider";
import { useDeploymentStatus } from "./use-deployment-status";

const SUCCESS_BANNER_MS = 4_500;

export type DeploymentPage = ReturnType<typeof useDeploymentPage>;

export function useDeploymentPage() {
  const buildRequested = Boolean(useSearchParams().get("build"));
  const { deployment } = useDeployment();
  const {
    getDomainsForDeployment,
    customDomains,
    isDomainsLoading,
    isCustomDomainsLoading,
    refetchDomains,
  } = useProjectData();
  const { app } = useAppCurrentDeployment();
  const { steps, derivedStatus } = useDeploymentStatus(deployment, { loadEndedSteps: true });
  const showSuccessBanner = useSuccessBanner(deployment.status);

  const ready = deployment.status === "ready";
  useEffect(() => {
    if (ready) {
      steps.refetch();
      refetchDomains();
    }
  }, [ready, refetchDomains, steps.refetch]);

  useEffect(() => {
    if (buildRequested) {
      const url = new URL(window.location.href);
      url.searchParams.delete("build");
      window.history.replaceState(null, "", url.href);
    }
  }, [buildRequested]);

  const domainsLoading = isDomainsLoading || isCustomDomainsLoading;
  return {
    deployment,
    steps: steps.data,
    stepsFetched: steps.isFetchedAfterMount,
    domainsLoading,
    showSuccessBanner,
    ...deploymentPageModel({
      deployment,
      status: derivedStatus,
      buildRequested,
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
