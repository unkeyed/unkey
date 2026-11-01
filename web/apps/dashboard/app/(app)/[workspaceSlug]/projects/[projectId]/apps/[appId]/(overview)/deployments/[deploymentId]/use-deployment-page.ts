import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import { useApp, useCustomDomains, useDomains } from "@/lib/deploy-data";
import { useFlag } from "@/lib/flags/provider";
import { useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { getDeploymentDomains } from "../../../components/domain-priority";
import { deploymentPageModel } from "./deployment-page-model";
import { useDeployment } from "./layout-provider";
import { useDeploymentStatus } from "./use-deployment-status";

const SUCCESS_BANNER_MS = 4_500;

export type DeploymentPage = ReturnType<typeof useDeploymentPage>;

export function useDeploymentPage() {
  const buildRequested = Boolean(useSearchParams().get("build"));
  const { deployment } = useDeployment();
  const domains = useDomains();
  const customDomains = useCustomDomains();
  const { data: app } = useApp();
  const { steps, derivedStatus } = useDeploymentStatus(deployment, { loadEndedSteps: true });
  const showSuccessBanner = useSuccessBanner(deployment.status);

  const ready = deployment.status === "ready";
  useEffect(() => {
    if (ready) {
      steps.refetch();
      domains.refetch();
    }
  }, [ready, domains.refetch, steps.refetch]);

  useEffect(() => {
    if (buildRequested) {
      const url = new URL(window.location.href);
      url.searchParams.delete("build");
      window.history.replaceState(null, "", url.href);
    }
  }, [buildRequested]);

  const domainsLoading = domains.isLoading || customDomains.isLoading;
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
      domains: getDeploymentDomains(deployment, domains.data),
      domainsLoading,
      customDomains: customDomains.data,
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
