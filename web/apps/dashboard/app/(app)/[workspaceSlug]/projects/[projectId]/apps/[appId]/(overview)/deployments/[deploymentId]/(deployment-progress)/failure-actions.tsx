"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { type Deployment, useEnvironments } from "@/lib/deploy-data";
import { routes } from "@/lib/navigation/routes";
import { Button } from "@unkey/ui";
import Link from "next/link";
import { useState } from "react";
import { RedeployDialog } from "../../components/table/components/actions/redeploy-dialog";
import type { StageKey } from "../details/stages-model";
import { settingsTargetFor } from "./settings-hint";

export function FailureActions({
  deployment,
  stage,
  error,
}: {
  deployment: Deployment;
  stage: StageKey;
  error: string;
}) {
  const workspace = useWorkspaceNavigation();
  const { data: environments } = useEnvironments();
  const [redeployOpen, setRedeployOpen] = useState(false);
  const settingsTarget = settingsTargetFor(
    error,
    environments.find((e) => e.id === deployment.environmentId)?.slug,
  );
  return (
    <>
      {settingsTarget && (
        <Button
          variant="outline"
          size="sm"
          className="px-3"
          render={
            <Link
              href={routes.projects.apps.settings({
                workspaceSlug: workspace.slug,
                projectId: deployment.projectId,
                appId: deployment.appId,
                ...settingsTarget,
              })}
            />
          }
        >
          Go to settings
        </Button>
      )}
      {stage !== "build" && (
        <Button
          variant="outline"
          size="sm"
          className="px-3"
          render={
            <Link
              href={routes.projects.logs({
                workspaceSlug: workspace.slug,
                projectId: deployment.projectId,
                deploymentId: deployment.id,
              })}
            />
          }
        >
          Go to logs
        </Button>
      )}
      <Button variant="primary" size="sm" className="px-3" onClick={() => setRedeployOpen(true)}>
        Redeploy
      </Button>
      <RedeployDialog
        isOpen={redeployOpen}
        onClose={() => setRedeployOpen(false)}
        selectedDeployment={deployment}
      />
    </>
  );
}
