"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { match } from "@unkey/match";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useMemo } from "react";
import { DeploymentDomainsCard } from "../../../components/deployment-domains-card";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";
import { useProjectData } from "../../data-provider";
import { DeploymentApproval } from "./(deployment-progress)/deployment-approval";
import { DeploymentBuild } from "./(deployment-progress)/deployment-build";
import { DeploymentCancelled } from "./(deployment-progress)/deployment-cancelled";
import { DeploymentInfo } from "./(deployment-progress)/deployment-info";
import { DeploymentProgress } from "./(deployment-progress)/deployment-progress";
import { DeploymentSkipped } from "./(deployment-progress)/deployment-skipped";
import { DeploymentNetworkSection } from "./(overview)/components/sections/deployment-network-section";
import { deriveStatusFromSteps } from "./deployment-utils";
import { useDeployment } from "./layout-provider";

export default function DeploymentOverview() {
  const searchParams = useSearchParams();
  const build = searchParams.get("build");
  const { deployment } = useDeployment();
  const { refetchDomains } = useProjectData();
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const params = useParams<{ projectId: string }>();

  const ready = deployment.status === "ready";
  const skipped = deployment.status === "skipped";
  const superseded = deployment.status === "superseded";
  const cancelled = deployment.status === "cancelled";
  const awaitingApproval = deployment.status === "awaiting_approval";
  // Steps don't change until the user authorizes (awaiting_approval) or
  // until the deployment reaches a true terminal state — no point polling.
  const stepsAreStable = ready || skipped || superseded || cancelled || awaitingApproval;

  const stepsQuery = trpc.deploy.deployment.steps.useQuery(
    { deploymentId: deployment.id },
    {
      refetchInterval: stepsAreStable ? false : 1_000,
      refetchOnWindowFocus: false,
      enabled: !skipped && !superseded && !cancelled,
    },
  );

  const derivedStatus = useMemo(
    () =>
      skipped ? ("skipped" as const) : deriveStatusFromSteps(stepsQuery.data, deployment.status),
    [stepsQuery.data, deployment.status, skipped],
  );

  useEffect(() => {
    if (ready) {
      stepsQuery.refetch();
      refetchDomains();
    }
  }, [ready, refetchDomains, stepsQuery.refetch]);

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
        <DeploymentProgress stepsData={stepsQuery.data} />
      </div>
    ))
    .with("skipped", () => (
      <div key="skipped" className="animate-fade-slide-in">
        <DeploymentSkipped />
      </div>
    ))
    .with("superseded", () => (
      <div key="superseded" className="animate-fade-slide-in">
        <DeploymentCancelled
          deployment={deployment}
          stepsData={stepsQuery.data}
          reason="superseded"
        />
      </div>
    ))
    .with("cancelled", () => (
      <div key="cancelled" className="animate-fade-slide-in">
        <DeploymentCancelled
          deployment={deployment}
          stepsData={stepsQuery.data}
          reason="cancelled"
        />
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
        <DeploymentProgress stepsData={stepsQuery.data} />
      </div>
    ));

  return (
    <ProjectContentWrapper centered>
      <DeploymentInfo statusOverride={derivedStatus} />
      {view}
      <DeploymentApproval
        isOpen={awaitingApproval}
        onClose={() => router.push(`/${workspace.slug}/projects/${params.projectId}/deployments`)}
        deployment={deployment}
      />
    </ProjectContentWrapper>
  );
}
