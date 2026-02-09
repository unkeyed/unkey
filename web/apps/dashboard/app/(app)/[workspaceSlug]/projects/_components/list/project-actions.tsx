"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { useWorkspace } from "@/providers/workspace-provider";
import { Clone, Cloud, Gear, Heart, Layers3, Trash } from "@unkey/icons";

import { toast } from "@unkey/ui";
import type { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";
import { useRouter } from "next/navigation";
import type { PropsWithChildren } from "react";
import { DeleteProjectDialog } from "../dialogs/delete-project-dialog";

type ProjectActionsProps = {
  projectId: string;
  projectName: string;
};

export const ProjectActions = ({
  projectId,
  projectName,
  children,
}: PropsWithChildren<ProjectActionsProps>) => {
  const router = useRouter();
  const { workspace } = useWorkspace();
  // biome-ignore lint/style/noNonNullAssertion: This cannot be null
  const menuItems = getProjectActionItems(projectId, projectName, workspace?.slug!, router);

  return <TableActionPopover items={menuItems}>{children}</TableActionPopover>;
};

const getProjectActionItems = (
  projectId: string,
  projectName: string,
  workspaceSlug: string,
  router: AppRouterInstance,
): MenuItem[] => {
  return [
    {
      id: "favorite-project",
      label: "Add favorite",
      icon: <Heart iconSize="md-medium" />,
      onClick: () => {},
      divider: true,
      disabled: true,
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
      label: "View sentinel logs",
      icon: <Layers3 iconSize="md-regular" />,
      onClick: () => {
        router.push(`/${workspaceSlug}/projects/${projectId}/sentinel-logs`);
      },
    },
    {
      id: "view-deployment",
      label: "View deployments",
      icon: <Cloud iconSize="md-regular" />,
      onClick: () => {
        router.push(`/${workspaceSlug}/projects/${projectId}/deployments`);
      },
    },
    {
      id: "project-settings",
      label: "Project settings",
      icon: <Gear iconSize="md-medium" />,
      onClick: () => {
        router.push(`/${workspaceSlug}/projects/${projectId}/settings`);
      },
      divider: true,
    },
    {
      id: "delete-project",
      label: "Delete project",
      icon: <Trash iconSize="md-medium" />,
      ActionComponent: ({ isOpen, onClose }) => (
        <DeleteProjectDialog
          projectId={projectId}
          projectName={projectName}
          isOpen={isOpen}
          onClose={onClose}
        />
      ),
    },
  ];
};
