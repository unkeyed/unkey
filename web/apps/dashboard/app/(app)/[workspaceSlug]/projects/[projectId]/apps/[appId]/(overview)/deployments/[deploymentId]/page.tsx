"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { match } from "@unkey/match";
import { PageBody } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { DeploymentApproval } from "./(deployment-progress)/deployment-approval";
import { DeploymentBuild } from "./(deployment-progress)/deployment-build";
import { DeploymentCancelled } from "./(deployment-progress)/deployment-cancelled";
import { DeploymentSkipped } from "./(deployment-progress)/deployment-skipped";
import { FailedDeploymentBanner } from "./(deployment-progress)/failed-deployment-banner";
import { DeploymentSuccessBanner } from "./deployment-success-banner";
import { DeploymentDetailsCard } from "./details/deployment-details-card";
import { DeploymentStagesCard } from "./details/deployment-stages-card";
import { useDeploymentPage } from "./use-deployment-page";

export default function DeploymentOverview() {
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
          appScope={appScope}
          limitsUrl={routes.settings.limits({ workspaceSlug: workspace.slug })}
          deployment={deployment}
        />
      )}
      <DeploymentDetailsCard page={page} />
      {page.body === "stages" ? (
        <DeploymentStagesCard
          steps={steps}
          status={page.status}
          instances={page.instances}
          domainCount={page.domainsLoading ? null : page.domains.all.length}
        />
      ) : (
        <div key={page.body} className="animate-fade-slide-in">
          {match(page.body)
            .with("build", () => <DeploymentBuild />)
            .with("skipped", () => <DeploymentSkipped />)
            .with("superseded", "cancelled", (reason) => (
              <DeploymentCancelled deployment={deployment} stepsData={steps} reason={reason} />
            ))
            .exhaustive()}
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
