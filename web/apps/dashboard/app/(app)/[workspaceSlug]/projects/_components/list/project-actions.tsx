"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Clone, Gear, Layers3, Terminal } from "@unkey/icons";

import { toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import type { PropsWithChildren } from "react";

type ProjectActionsProps = {
  projectId: string;
};

export const ProjectActions = ({ projectId, children }: PropsWithChildren<ProjectActionsProps>) => {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  const menuItems: MenuItem[] = [
    {
      id: "copy-project-id",
      label: "Copy project ID",
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
      id: "view-requests",
      label: "View requests",
      icon: <Layers3 iconSize="md-regular" />,
      onClick: () => {
        router.push(`/${workspace.slug}/projects/${projectId}/requests`);
      },
    },
    {
      id: "view-logs",
      label: "View logs",
      icon: <Terminal iconSize="md-regular" />,
      onClick: () => {
        router.push(`/${workspace.slug}/projects/${projectId}/logs`);
      },
    },
    {
      id: "project-settings",
      label: "Project settings",
      icon: <Gear iconSize="md-medium" />,
      onClick: () => {
        router.push(`/${workspace.slug}/projects/${projectId}/settings`);
      },
    },
  ];

  return <TableActionPopover items={menuItems}>{children}</TableActionPopover>;
};
