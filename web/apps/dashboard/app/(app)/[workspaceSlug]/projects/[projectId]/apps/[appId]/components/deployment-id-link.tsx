"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { deploymentPath } from "@/lib/navigation/routes/projects";
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
  // no appId means the deployment is not in the loaded collection, show the id without a broken link
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
