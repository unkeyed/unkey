"use client";

import {
  type MenuItem,
  TableActionPopover,
} from "@/components/logs/table-action.popover";
import { useWorkspace } from "@/providers/workspace-provider";
import { Clone, Gear, Layers3, Trash } from "@unkey/icons";

import { toast } from "@unkey/ui";
import type { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";
import { useRouter } from "next/navigation";
import type { PropsWithChildren } from "react";

type ProjectActionsProps = {
  projectId: string;
};

export const ProjectActions = ({
  projectId,
  children,
}: PropsWithChildren<ProjectActionsProps>) => {
  const router = useRouter();
  const { workspace } = useWorkspace();
  // biome-ignore lint/style/noNonNullAssertion: This cannot be null
  const menuItems = getProjectActionItems(projectId, workspace?.slug!, router);

  return <TableActionPopover items={menuItems}>{children}</TableActionPopover>;
};

const getProjectActionItems = (
  projectId: string,
  workspaceSlug: string,
  router: AppRouterInstance
): MenuItem[] => {
  return [
    {
      id: "favorite-project",
      label: "Add favorite",
      icon: <Gear iconSize="md-medium" />,
      onClick: () => {},
      divider: true,
    },
    {
      id: "copy-project-id",
      label: "Copy project ID",
      className: "mt-1",
      icon: <Clone iconSize="md-medium" />,
      onClick: () => {
        navigator.clipboard
          .writeText(projectId)
          .then(() => {
            toast.success("Project ID copied to clipboard");
          })
          .catch((error) => {
            console.error("Failed to copy to clipboard:", error);
            toast.error("Failed to copy to clipboard");
          });
      },
      divider: true,
    },
    {
      id: "view-log",
      label: "View gateway logs",
      icon: <Layers3 iconSize="md-regular" />,
      onClick: () => {
        router.push(`/${workspaceSlug}/projects/${projectId}/gateway-logs`);
      },
    },
    {
      id: "project-settings",
      label: "Project settings",
      icon: <Gear iconSize="md-medium" />,
      onClick: () => {
        //INFO: This will change soon
        const fakeDeploymentId = "idk";
        router.push(
          `/projects/${projectId}/deployments/${fakeDeploymentId}/settings`
        );
      },
      divider: true,
    },
    {
      id: "delete-project",
      label: "Delete project",
      icon: <Trash iconSize="md-medium" />,
      ActionComponent: () => null,
    },
  ];
};
