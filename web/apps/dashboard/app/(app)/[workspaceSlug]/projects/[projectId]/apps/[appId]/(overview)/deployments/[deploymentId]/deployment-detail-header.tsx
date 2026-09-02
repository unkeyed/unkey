"use client";

import {
  Button,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import dynamic from "next/dynamic";
import {
  IconArrowDottedRotateAnticlockwiseOutline18,
  IconBanOutline18,
  IconDotsOutline18,
} from "nucleo-ui-outline-18";
import { useState } from "react";
import { TableActionPopover } from "@/components/logs/table-action.popover";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import { shortenId } from "@/lib/shorten-id";
import { useProjectData } from "../../data-provider";
import {
  isCancellableDeploymentStatus,
  isRedeployableDeploymentStatus,
} from "../components/table/components/actions/deployment-action-eligibility";
import { useDeploymentHeaderActions } from "./deployment-header-actions";
import { useDeployment } from "./layout-provider";
import { useDeploymentStatus } from "./use-deployment-status";

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
  const { environments } = useProjectData();

  const { derivedStatus } = useDeploymentStatus(deployment);
  const [isRedeployOpen, setIsRedeployOpen] = useState(false);
  const [isCancelOpen, setIsCancelOpen] = useState(false);
  const [cancelled, setCancelled] = useState(false);
  const canCancel = isCancellableDeploymentStatus(derivedStatus) && !cancelled;
  const canRedeploy = isRedeployableDeploymentStatus(derivedStatus);

  const title = deployment.gitCommitMessage || shortenId(deployment.id);
  const environment = environments.find((env) => env.id === deployment.environmentId);

  const { items, planGate, gated, openPaywall } = useDeploymentHeaderActions({
    deployment,
    environment,
    status: derivedStatus,
  });

  return (
    <PageHeader>
      <PageHeaderContent>
        <PageHeaderTitle className="truncate" title={title}>
          {title}
        </PageHeaderTitle>
      </PageHeaderContent>
      <PageHeaderActions>
        <TableActionPopover items={items}>
          <Button variant="outline" className="w-7 p-0" aria-label="Open actions">
            <IconDotsOutline18 />
          </Button>
        </TableActionPopover>
        {canCancel && (
          <Button variant="outline" onClick={() => setIsCancelOpen(true)}>
            <IconBanOutline18 />
            Cancel deployment
          </Button>
        )}
        {canRedeploy && (
          <Button
            variant="outline"
            onClick={() => (gated ? openPaywall() : setIsRedeployOpen(true))}
          >
            <IconArrowDottedRotateAnticlockwiseOutline18 />
            Redeploy
          </Button>
        )}
      </PageHeaderActions>
      {canRedeploy && (
        <RedeployDialog
          isOpen={isRedeployOpen}
          onClose={() => setIsRedeployOpen(false)}
          selectedDeployment={deployment}
        />
      )}
      {canCancel && (
        <CancelDialog
          isOpen={isCancelOpen}
          onClose={() => setIsCancelOpen(false)}
          onCancelled={() => setCancelled(true)}
          deployment={deployment}
        />
      )}
      {planGate}
    </PageHeader>
  );
}
