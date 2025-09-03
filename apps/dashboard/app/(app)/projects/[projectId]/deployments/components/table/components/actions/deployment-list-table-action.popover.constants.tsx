"use client";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { Deployment } from "@/lib/trpc/routers/deploy/project/deployment/list";
import { ArrowDottedRotateAnticlockwise, PenWriting3 } from "@unkey/icons";
import type { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { RollbackDialog } from "../../../rollback-dialog";

type DeploymentListTableActionsProps = {
  deployment: Deployment;
  currentActiveDeployment?: Deployment & { environment: "production"; active: true };
};

export const DeploymentListTableActions = ({ deployment, currentActiveDeployment }: DeploymentListTableActionsProps) => {
  const router = useRouter();
  const [isRollbackModalOpen, setIsRollbackModalOpen] = useState(false);
  const menuItems = getDeploymentListTableActionItems(deployment, router, setIsRollbackModalOpen);
  
  return (
    <>
      <TableActionPopover items={menuItems} />
      <RollbackDialog
        isOpen={isRollbackModalOpen}
        onOpenChange={setIsRollbackModalOpen}
        deployment={deployment}
        currentDeployment={currentActiveDeployment}
        hostname="example.com" // TODO: Get actual hostname from deployment/project
      />
    </>
  );
};

const getDeploymentListTableActionItems = (
  deployment: Deployment,
  router: AppRouterInstance,
  setIsRollbackModalOpen: (open: boolean) => void,
): MenuItem[] => {
  // Rollback is only enabled for production deployments that are completed and not currently active
  const canRollback = 
    deployment.environment === "production" &&
    deployment.status === "completed" &&
    !("active" in deployment && deployment.active);

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
