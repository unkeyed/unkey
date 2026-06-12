"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import { shortenId } from "@/lib/shorten-id";
import {
  ArrowDottedRotateAnticlockwise,
  ArrowOppositeDirectionY,
  Ban,
  Layers3,
} from "@unkey/icons";
import {
  Button,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useState } from "react";
import { isCancellableDeploymentStatus } from "../components/table/components/actions/deployment-action-eligibility";
import { useDeployment } from "./layout-provider";

const RedeployDialog = dynamic(
  () =>
    import("../components/table/components/actions/redeploy-dialog").then((m) => m.RedeployDialog),
  { ssr: false },
);

const CancelDialog = dynamic(
  () => import("../components/table/components/actions/cancel-dialog").then((m) => m.CancelDialog),
  { ssr: false },
);

export function DeploymentDetailHeader() {
  const { deployment } = useDeployment();
  // Keyed by id so dialog and cancelled state reset when navigation swaps
  // the deployment under this layout (e.g. Redeploy pushes to the new one).
  return <DeploymentDetailHeaderContent key={deployment.id} deployment={deployment} />;
}

function DeploymentDetailHeaderContent({ deployment }: { deployment: Deployment }) {
  const workspace = useWorkspaceNavigation();

  const [isRedeployOpen, setIsRedeployOpen] = useState(false);
  const [isCancelOpen, setIsCancelOpen] = useState(false);
  const [cancelled, setCancelled] = useState(false);
  const canCancel = isCancellableDeploymentStatus(deployment.status) && !cancelled;

  const title = deployment.gitCommitMessage || shortenId(deployment.id);
  const subtitle = [deployment.gitBranch, deployment.gitCommitSha?.slice(0, 7)]
    .filter(Boolean)
    .join(" · ");

  const projectBase = `/${workspace.slug}/projects/${deployment.projectId}`;
  const deploymentFilter = `deploymentId=is:${deployment.id}`;

  return (
    <PageHeader>
      <PageHeaderContent>
        <PageHeaderTitle className="truncate" title={title}>
          {title}
        </PageHeaderTitle>
        {subtitle && (
          <PageHeaderDescription className="font-mono">{subtitle}</PageHeaderDescription>
        )}
      </PageHeaderContent>
      <PageHeaderActions>
        <Button variant="outline" asChild>
          <Link href={`${projectBase}/logs?${deploymentFilter}`}>
            <Layers3 iconSize="sm-regular" />
            Logs
          </Link>
        </Button>
        <Button variant="outline" asChild>
          <Link href={`${projectBase}/requests?${deploymentFilter}`}>
            <ArrowOppositeDirectionY iconSize="sm-regular" />
            Requests
          </Link>
        </Button>
        {canCancel && (
          <Button variant="outline" onClick={() => setIsCancelOpen(true)}>
            <Ban iconSize="sm-regular" />
            Cancel deployment
          </Button>
        )}
        <Button variant="outline" onClick={() => setIsRedeployOpen(true)}>
          <ArrowDottedRotateAnticlockwise iconSize="sm-regular" />
          Redeploy
        </Button>
      </PageHeaderActions>
      <RedeployDialog
        isOpen={isRedeployOpen}
        onClose={() => setIsRedeployOpen(false)}
        selectedDeployment={deployment}
      />
      {canCancel && (
        <CancelDialog
          isOpen={isCancelOpen}
          onClose={() => setIsCancelOpen(false)}
          onCancelled={() => setCancelled(true)}
          deployment={deployment}
        />
      )}
    </PageHeader>
  );
}
