"use client";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { Deployment } from "@/lib/collections";
import { PenWriting3 } from "@unkey/icons";
import type { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";
import { useRouter } from "next/navigation";

type DeploymentListTableActionsProps = {
  deployment: Deployment;
};

export const DeploymentListTableActions = ({ deployment }: DeploymentListTableActionsProps) => {
  const router = useRouter();
  const menuItems = getDeploymentListTableActionItems(deployment, router);
  return <TableActionPopover items={menuItems} />;
};

const getDeploymentListTableActionItems = (
  deployment: Deployment,
  router: AppRouterInstance,
): MenuItem[] => {
  return [
    {
      id: "edit-root-key",
      label: "Edit root key...",
      icon: <PenWriting3 size="md-regular" />,
      onClick: () => {
        router.push(`/settings/root-keys/${deployment.id}`);
      },
    },
  ];
};
