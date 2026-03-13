"use client";
import { trpc } from "@/lib/trpc/client";
import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { DeploymentDomainsCard } from "../../../components/deployment-domains-card";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";
import { useProjectData } from "../../data-provider";
import { DeploymentInfo } from "./(deployment-progress)/deployment-info";
import { FailedDeploymentBanner } from "./(deployment-progress)/failed-deployment-banner";
import { DeploymentProgress } from "./(deployment-progress)/deployment-progress";
import { SkippedDeploymentView } from "./(deployment-progress)/skipped-deployment-view";
import { DeploymentNetworkSection } from "./(overview)/components/sections/deployment-network-section";
import { deriveStatusFromSteps } from "./deployment-utils";
import { useDeployment } from "./layout-provider";

export default function DeploymentOverview() {
  const { deployment } = useDeployment();
  const { refetchDomains } = useProjectData();

  const ready = deployment.status === "ready";
  const skipped = deployment.status === "skipped";

  const stepsQuery = trpc.deploy.deployment.steps.useQuery(
    { deploymentId: deployment.id },
    {
      refetchInterval: ready || skipped ? false : 1_000,
      refetchOnWindowFocus: false,
      enabled: !skipped,
    },
  );

  const derivedStatus = useMemo(
    () =>
      skipped ? ("skipped" as const) : deriveStatusFromSteps(stepsQuery.data, deployment.status),
    [stepsQuery.data, deployment.status, skipped],
  );

  // A post-deploy failure is when the pipeline completed successfully but the
  // deployment was later marked as failed (e.g. all instances crashed).
  const pipelineCompleted = stepsQuery.data?.finalizing?.completed === true;
  const isPostDeployFailure = derivedStatus === "failed" && pipelineCompleted;

  const [redeployOpen, setRedeployOpen] = useState(false);
  const params = useParams();
  const workspaceSlug = params.workspaceSlug as string;
  const { projectId } = useProjectData();

  // Fetch instance errors for post-deploy failure banner
  const { data: deploymentTree } = trpc.deploy.network.get.useQuery(
    { deploymentId: deployment.id },
    { enabled: isPostDeployFailure },
  );
  const instanceErrors = isPostDeployFailure
    ? (deploymentTree?.children ?? [])
        .flatMap((sentinel) => ("children" in sentinel && sentinel.children) ? sentinel.children : [])
        .filter((inst) => inst.metadata.type === "instance" && "message" in inst.metadata && inst.metadata.message)
        .map((inst) => (inst.metadata as { message: string }).message)
        .filter((msg, i, arr) => arr.indexOf(msg) === i)
    : [];

  useEffect(() => {
    if (ready) {
      stepsQuery.refetch();
      refetchDomains();
    }
  }, [ready, refetchDomains, stepsQuery.refetch]);

  return (
    <ProjectContentWrapper centered>
      <DeploymentInfo statusOverride={derivedStatus} />
      {skipped ? (
        <div key="skipped" className="animate-fade-slide-in">
          <SkippedDeploymentView />
        </div>
      ) : ready || isPostDeployFailure ? (
        <div key="ready" className="flex flex-col gap-5 animate-fade-slide-in">
          {isPostDeployFailure && (
            <FailedDeploymentBanner
              steps={[]}
              settingsUrl={`/${workspaceSlug}/projects/${projectId}/settings`}
              onRedeploy={() => setRedeployOpen(true)}
              redeployOpen={redeployOpen}
              onRedeployClose={() => setRedeployOpen(false)}
              deployment={deployment}
              instanceErrors={instanceErrors}
            />
          )}
          <DeploymentDomainsCard />
          <DeploymentNetworkSection />
        </div>
      ) : (
        <div key="progress" className="animate-fade-slide-in">
          <DeploymentProgress stepsData={stepsQuery.data} />
        </div>
      )}
    </ProjectContentWrapper>
  );
}
