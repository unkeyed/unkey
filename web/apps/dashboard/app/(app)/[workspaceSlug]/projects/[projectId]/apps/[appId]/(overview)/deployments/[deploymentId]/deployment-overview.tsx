"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { PageBody } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { DeploymentApproval } from "./(deployment-progress)/deployment-approval";
import { DeploymentSkipped } from "./(deployment-progress)/deployment-skipped";
import { FailedDeploymentBanner } from "./(deployment-progress)/failed-deployment-banner";
import { ConnectionPinsNotice } from "./connection-pins-notice";
import { DeploymentSuccessBanner } from "./deployment-success-banner";
import { DeploymentDetailsCard } from "./details/deployment-details-card";
import { DeploymentStagesCard } from "./details/deployment-stages-card";
import { useDeploymentPage } from "./use-deployment-page";

export function DeploymentOverview() {
  const page = useDeploymentPage();
  const { deployment, steps } = page;
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const appScope = {
    workspaceSlug: workspace.slug,
    projectId: deployment.projectId,
    appId: deployment.appId,
  };

  return (
    <PageBody>
      <DeploymentSuccessBanner visible={page.showSuccessBanner} />
      {page.status === "failed" && (
        <FailedDeploymentBanner
          stepsData={steps}
          limitsUrl={routes.settings.limits({ workspaceSlug: workspace.slug })}
        />
      )}
      <DeploymentDetailsCard page={page} />
      <ConnectionPinsNotice />
      {page.body === "stages" ? (
        <DeploymentStagesCard page={page} />
      ) : (
        <div className="animate-fade-slide-in">
          <DeploymentSkipped />
        </div>
      )}
      <DeploymentApproval
        isOpen={page.status === "awaiting_approval"}
        onClose={() => router.push(routes.projects.apps.deployments(appScope))}
        deployment={deployment}
      />
    </PageBody>
  );
}
