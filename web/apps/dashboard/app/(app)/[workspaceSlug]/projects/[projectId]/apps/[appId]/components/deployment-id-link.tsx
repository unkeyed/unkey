"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { deploymentPath } from "@/lib/navigation/routes";
import { shortenId } from "@/lib/shorten-id";
import { CopyButton } from "@unkey/ui";
import { useProjectData } from "../(overview)/data-provider";
import { DottedLink } from "./dotted-link";

type DeploymentIdLinkProps = {
  deploymentId: string;
};

export function DeploymentIdLink({ deploymentId }: DeploymentIdLinkProps) {
  const workspace = useWorkspaceNavigation();
  const { projectId, getDeploymentById } = useProjectData();
  // appId comes from the deployment record, not the route: project-level logs and
  // requests render rows from many apps. A miss (deployment outside the loaded
  // window) leaves no appId, so show the id without a link instead of crashing.
  const appId = getDeploymentById(deploymentId)?.appId;

  if (!appId) {
    return (
      <div className="flex items-center gap-2">
        <span className="font-mono text-xs">{shortenId(deploymentId)}</span>
        <CopyButton value={deploymentId} variant="ghost" className="h-4 w-4" />
      </div>
    );
  }

  return (
    <DottedLink
      href={deploymentPath({ workspaceSlug: workspace.slug, projectId, appId, deploymentId })}
      copyValue={deploymentId}
    >
      <span className="font-mono">{shortenId(deploymentId)}</span>
    </DottedLink>
  );
}
