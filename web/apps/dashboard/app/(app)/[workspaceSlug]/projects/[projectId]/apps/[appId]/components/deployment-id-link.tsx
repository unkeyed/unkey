"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useDeployScope, useDeployment } from "@/lib/deploy-data";
import { routes } from "@/lib/navigation/routes";
import { shortenId } from "@/lib/shorten-id";
import { CopyButton } from "@unkey/ui";
import { DottedLink } from "./dotted-link";

type DeploymentIdLinkProps = {
  deploymentId: string;
};

export function DeploymentIdLink({ deploymentId }: DeploymentIdLinkProps) {
  const workspace = useWorkspaceNavigation();
  const { projectId } = useDeployScope();
  // Without a row there is no app to link into, so the id shows without a broken link
  const appId = useDeployment(deploymentId).data?.appId;

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
      href={routes.projects.apps.deployment({
        workspaceSlug: workspace.slug,
        projectId,
        appId,
        deploymentId,
      })}
      copyValue={deploymentId}
    >
      <span className="font-mono">{shortenId(deploymentId)}</span>
    </DottedLink>
  );
}
