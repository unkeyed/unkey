"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import { routes } from "@/lib/navigation/routes";
import { Button } from "@unkey/ui";
import Link from "next/link";
import { useState } from "react";
import { RedeployDialog } from "../../components/table/components/actions/redeploy-dialog";

export function FailureActions({
  deployment,
  redeployVariant = "primary",
}: {
  deployment: Deployment;
  redeployVariant?: "primary" | "outline";
}) {
  const workspace = useWorkspaceNavigation();
  const [redeployOpen, setRedeployOpen] = useState(false);
  const logsHref = routes.projects.logs({
    workspaceSlug: workspace.slug,
    projectId: deployment.projectId,
    deploymentId: deployment.id,
  });
  return (
    <>
      <Button variant="outline" size="sm" className="px-3" render={<Link href={logsHref} />}>
        Go to logs
      </Button>
      <Button
        variant={redeployVariant}
        size="sm"
        className="px-3"
        onClick={() => setRedeployOpen(true)}
      >
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
