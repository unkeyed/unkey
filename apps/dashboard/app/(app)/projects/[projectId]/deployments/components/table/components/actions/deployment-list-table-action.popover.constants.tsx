"use client";
import { TableActionPopover } from "@/components/logs/table-action.popover";
import type { Deployment, Environment } from "@/lib/collections";
import { ArrowDottedRotateAnticlockwise, ChevronUp } from "@unkey/icons";
import { useState } from "react";
import { PromotionDialog } from "../../../promotion-dialog";
import { RollbackDialog } from "../../../rollback-dialog";

type DeploymentListTableActionsProps = {
  liveDeployment?: Deployment;
  selectedDeployment: Deployment;
  environment?: Environment;
};

export const DeploymentListTableActions = ({
  liveDeployment,
  selectedDeployment,
  environment,
}: DeploymentListTableActionsProps) => {
  const [isRollbackModalOpen, setIsRollbackModalOpen] = useState(false);
  const [isPromotionModalOpen, setIsPromotionModalOpen] = useState(false);

  const canRollback =
    liveDeployment &&
    environment?.slug === "production" &&
    selectedDeployment.status === "ready" &&
    selectedDeployment.id !== liveDeployment.id;

  // TODO
  // This logic is slightly flawed as it does not allow you to promote a deployment that
  // is currently live due to a rollback.
  const canPromote =
    liveDeployment &&
    environment?.slug === "production" &&
    selectedDeployment.status === "ready" &&
    selectedDeployment.id !== liveDeployment.id;

  return (
    <>
      <TableActionPopover
        items={[
          {
            id: "rollback",
            label: "Rollback",
            icon: <ArrowDottedRotateAnticlockwise size="md-regular" />,
            disabled: !canRollback,
            onClick: () => {
              if (canRollback) {
                setIsRollbackModalOpen(true);
              }
            },
          },
          {
            id: "Promote",
            label: "Promote",
            icon: <ChevronUp size="md-regular" />,
            disabled: !canPromote,
            onClick: () => {
              if (canPromote) {
                setIsPromotionModalOpen(true);
              }
            },
          },
        ]}
      />
      {liveDeployment && selectedDeployment && (
        <RollbackDialog
          isOpen={isRollbackModalOpen}
          onOpenChange={setIsRollbackModalOpen}
          liveDeployment={liveDeployment}
          targetDeployment={selectedDeployment}
        />
      )}
      {liveDeployment && selectedDeployment && (
        <PromotionDialog
          isOpen={isPromotionModalOpen}
          onOpenChange={setIsPromotionModalOpen}
          liveDeployment={liveDeployment}
          targetDeployment={selectedDeployment}
        />
      )}
    </>
  );
};
