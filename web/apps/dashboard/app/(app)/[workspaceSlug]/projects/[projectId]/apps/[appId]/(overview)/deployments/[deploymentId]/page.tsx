"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { match } from "@unkey/match";
import { PageBody } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect } from "react";
import { DeploymentDomainsCard } from "../../../components/deployment-domains-card";
import { useProjectData } from "../../data-provider";
import { DeploymentApproval } from "./(deployment-progress)/deployment-approval";
import { DeploymentBuild } from "./(deployment-progress)/deployment-build";
import { DeploymentCancelled } from "./(deployment-progress)/deployment-cancelled";
import { DeploymentInfo } from "./(deployment-progress)/deployment-info";
import { DeploymentProgress } from "./(deployment-progress)/deployment-progress";
import { DeploymentSkipped } from "./(deployment-progress)/deployment-skipped";
import { DeploymentNetworkSection } from "./(overview)/components/sections/deployment-network-section";
import { useDeployment } from "./layout-provider";
import { useDeploymentStatus } from "./use-deployment-status";

export default function DeploymentOverview() {
  const searchParams = useSearchParams();
  const build = searchParams.get("build");
  const { deployment } = useDeployment();
  const { refetchDomains } = useProjectData();
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  const ready = deployment.status === "ready";
  const awaitingApproval = deployment.status === "awaiting_approval";

  const { steps, derivedStatus } = useDeploymentStatus(deployment);

  useEffect(() => {
    if (ready) {
      steps.refetch();
      refetchDomains();
    }
  }, [ready, refetchDomains, steps.refetch]);

  const view = match(deployment.status)
    .when(
      () => Boolean(build),
      () => (
        <div key="build" className="animate-fade-slide-in">
          <DeploymentBuild />
        </div>
      ),
    )
    .with("awaiting_approval", () => (
      <div key="approval" className="animate-fade-slide-in">
        <DeploymentProgress stepsData={steps.data} />
      </div>
    ))
    .with("skipped", () => (
      <div key="skipped" className="animate-fade-slide-in">
        <DeploymentSkipped />
      </div>
    ))
    .with("superseded", () => (
      <div key="superseded" className="animate-fade-slide-in">
        <DeploymentCancelled deployment={deployment} stepsData={steps.data} reason="superseded" />
      </div>
    ))
    .with("cancelled", () => (
      <div key="cancelled" className="animate-fade-slide-in">
        <DeploymentCancelled deployment={deployment} stepsData={steps.data} reason="cancelled" />
      </div>
    ))
    .with("ready", () => (
      <div key="ready" className="flex flex-col gap-5 animate-fade-slide-in">
        <DeploymentDomainsCard />
        <DeploymentNetworkSection />
      </div>
    ))
    .otherwise(() => (
      <div key="progress" className="animate-fade-slide-in">
        <DeploymentProgress stepsData={steps.data} />
      </div>
    ));

  return (
    <PageBody>
      <DeploymentInfo statusOverride={derivedStatus} />
      {view}
      <DeploymentApproval
        isOpen={awaitingApproval}
        onClose={() =>
          router.push(
            routes.projects.apps.deployments({
              workspaceSlug: workspace.slug,
              projectId: deployment.projectId,
              appId: deployment.appId,
            }),
          )
        }
        deployment={deployment}
      />
    </PageBody>
  );
}
