"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { shortenId } from "@/lib/shorten-id";
import { useProjectData } from "../(overview)/data-provider";
import { DottedLink } from "./dotted-link";

type DeploymentIdLinkProps = {
  deploymentId: string;
};

export function DeploymentIdLink({ deploymentId }: DeploymentIdLinkProps) {
  const workspace = useWorkspaceNavigation();
  const { projectId } = useProjectData();

  return (
    <DottedLink
      href={`/${workspace.slug}/projects/${projectId}/deployments/${deploymentId}`}
      copyValue={deploymentId}
    >
      <span className="font-mono">{shortenId(deploymentId)}</span>
    </DottedLink>
  );
}
