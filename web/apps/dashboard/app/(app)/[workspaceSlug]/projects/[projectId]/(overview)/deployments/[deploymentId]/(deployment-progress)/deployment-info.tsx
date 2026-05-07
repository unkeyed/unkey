"use client";

import type { DeploymentStatus } from "@/lib/collections";
import { CircleXMark, Cloud } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { ActiveDeploymentCard } from "../../../../components/active-deployment-card";
import { DeploymentStatusBadge } from "../../../../components/deployment-status-badge";
import { Section, SectionHeader } from "../../../../components/section";
import { useProjectData } from "../../../data-provider";
import { CancelDialog } from "../../components/table/components/actions/cancel-dialog";
import { isCancellableDeploymentStatus } from "../../components/table/components/actions/deployment-action-eligibility";
import { useDeployment } from "../layout-provider";

type DeploymentInfoProps = {
  title?: string;
  statusOverride?: DeploymentStatus;
};

export function DeploymentInfo({ title = "Deployment", statusOverride }: DeploymentInfoProps) {
  const { deployment } = useDeployment();
  const { project, environments } = useProjectData();
  const deploymentStatus = statusOverride ?? deployment.status;
  // Track "the cancel mutation already succeeded for this view" so the
  // button hides immediately even if the live query hasn't caught up yet.
  const [cancelled, setCancelled] = useState(false);
  const canCancel = isCancellableDeploymentStatus(deploymentStatus) && !cancelled;
  const [cancelOpen, setCancelOpen] = useState(false);

  const isCurrent = project?.currentDeploymentId === deployment.id;
  const isRolledBack = isCurrent && (project?.isRolledBack ?? false);
  const environment = environments.find((e) => e.id === deployment.environmentId);

  return (
    <Section>
      <SectionHeader
        icon={<Cloud iconSize="md-regular" className="text-gray-9" />}
        title={title}
        rightAction={
          canCancel ? (
            <Button
              variant="outline"
              size="sm"
              className="font-medium"
              onClick={() => setCancelOpen(true)}
            >
              <CircleXMark iconSize="sm-medium" />
              Cancel
            </Button>
          ) : undefined
        }
      />
      <ActiveDeploymentCard
        deploymentId={deployment.id}
        deployment={deployment}
        isCurrent={isCurrent}
        isRolledBack={isRolledBack}
        environmentSlug={environment?.slug}
        statusBadge={<DeploymentStatusBadge status={deploymentStatus} />}
      />
      {canCancel && (
        <CancelDialog
          isOpen={cancelOpen}
          onClose={() => setCancelOpen(false)}
          onCancelled={() => setCancelled(true)}
          deployment={deployment}
        />
      )}
    </Section>
  );
}
