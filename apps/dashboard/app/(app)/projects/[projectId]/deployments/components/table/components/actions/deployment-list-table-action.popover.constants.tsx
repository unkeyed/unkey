"use client";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { Deployment, Environment } from "@/lib/collections";
import { ArrowDottedRotateAnticlockwise, PenWriting3 } from "@unkey/icons";
import type { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { RollbackDialog } from "../../../rollback-dialog";

type DeploymentListTableActionsProps = {
  deployment: Deployment;
  currentActiveDeployment?: Deployment;
  environment?: Environment;
};

export const DeploymentListTableActions = ({ deployment, currentActiveDeployment, environment }: DeploymentListTableActionsProps) => {
  const router = useRouter();
  const [isRollbackModalOpen, setIsRollbackModalOpen] = useState(false);
  const menuItems = getDeploymentListTableActionItems(deployment, environment, router, setIsRollbackModalOpen);

  return (
    <>
      <TableActionPopover items={menuItems} />
      {currentActiveDeployment && (
        <RollbackDialog
          isOpen={isRollbackModalOpen}
          onOpenChange={setIsRollbackModalOpen}
          deployment={deployment}
          currentDeployment={currentActiveDeployment}
          hostname="example.com" // TODO: Get actual hostname from deployment/project
        />
      )}
    </>
  );
};

const getDeploymentListTableActionItems = (
  deployment: Deployment,
  environment: Environment | undefined,
  router: AppRouterInstance,
  setIsRollbackModalOpen: (open: boolean) => void,
): MenuItem[] => {
  // Rollback is only enabled for production deployments that are ready and not currently active
  const canRollback =
    environment?.slug === "production" &&
    deployment.status === "ready" &&
    deployment.id !== "current_active_deployment_id"; // TODO: Better way to determine if this is the current active deployment

  return [
    {
      id: "edit-root-key",
      label: "Edit root key...",
      icon: <PenWriting3 size="md-regular" />,
      onClick: () => {
        router.push(`/settings/root-keys/${deployment.id}`);
      },
    },
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
  ];
};
