"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
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
  const appId = getDeploymentById(deploymentId)?.appId;

  // no appId means the deployment is not in the loaded collection, show the id without a broken link
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
      href={`/${workspace.slug}/projects/${projectId}/apps/${appId}/deployments/${deploymentId}`}
      copyValue={deploymentId}
    >
      <span className="font-mono">{shortenId(deploymentId)}</span>
    </DottedLink>
  );
}
