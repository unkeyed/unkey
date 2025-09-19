"use client";
import {
  type MenuItem,
  TableActionPopover,
} from "@/components/logs/table-action.popover";
import type { Deployment, Environment } from "@/lib/collections";
import { ArrowDottedRotateAnticlockwise } from "@unkey/icons";
import { useState } from "react";
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
  const menuItems = getDeploymentListTableActionItems(
    selectedDeployment,
    liveDeployment,
    environment,
    setIsRollbackModalOpen
  );

  return (
    <>
      <TableActionPopover items={menuItems} />
      {liveDeployment && selectedDeployment && (
        <RollbackDialog
          isOpen={isRollbackModalOpen}
          onOpenChange={setIsRollbackModalOpen}
          liveDeployment={liveDeployment}
          targetDeployment={selectedDeployment}
        />
      )}
    </>
  );
};

const getDeploymentListTableActionItems = (
  selectedDeployment: Deployment,
  liveDeployment: Deployment | undefined,
  environment: Environment | undefined,
  setIsRollbackModalOpen: (open: boolean) => void
): MenuItem[] => {
  // Rollback is only enabled for production deployments that are ready and not currently active
  const canRollback =
    liveDeployment &&
    environment?.slug === "production" &&
    selectedDeployment.status === "ready" &&
    selectedDeployment.id !== liveDeployment.id;

  return [
    {
      id: "rollback",
      label: "Rollback",
      icon: <ArrowDottedRotateAnticlockwise iconsize="md-medium" />,
      disabled: !canRollback,
      onClick: () => {
        if (canRollback) {
          setIsRollbackModalOpen(true);
        }
      },
    },
  ];
};
