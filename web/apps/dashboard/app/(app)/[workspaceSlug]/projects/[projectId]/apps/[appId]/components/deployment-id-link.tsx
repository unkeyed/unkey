"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { PRODUCTION_ENVIRONMENT_SLUG } from "@/lib/collections/deploy/environments";
import { routes } from "@/lib/navigation/routes";
import { shortenId } from "@/lib/shorten-id";
import { CopyButton } from "@unkey/ui";
import { useProjectData } from "../data-provider";
import { DottedLink } from "./dotted-link";

type DeploymentIdLinkProps = {
  deploymentId: string;
};

export function DeploymentIdLink({ deploymentId }: DeploymentIdLinkProps) {
  const workspace = useWorkspaceNavigation();
  const { projectId, environments, getDeploymentById } = useProjectData();
  // An unloaded deployment has no app or environment to link into, so show the id alone.
  const deployment = getDeploymentById(deploymentId);

  if (!deployment) {
    return (
      <div className="flex items-center gap-2">
        <span className="font-mono text-xs">{shortenId(deploymentId)}</span>
        <CopyButton value={deploymentId} variant="ghost" className="h-4 w-4" />
      </div>
    );
  }

  return (
    <DottedLink
      href={routes.projects.apps.deployment({
        workspaceSlug: workspace.slug,
        projectId,
        appId: deployment.appId,
        environmentSlug:
          environments.find((e) => e.id === deployment.environmentId)?.slug ??
          PRODUCTION_ENVIRONMENT_SLUG,
        deploymentId,
      })}
      copyValue={deploymentId}
    >
      <span className="font-mono">{shortenId(deploymentId)}</span>
    </DottedLink>
  );
}
