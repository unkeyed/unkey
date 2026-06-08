"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { Clone, Heart } from "@unkey/icons";

import { toast } from "@unkey/ui";
import type { PropsWithChildren } from "react";

type ProjectActionsProps = {
  projectId: string;
};

export const ProjectActions = ({ projectId, children }: PropsWithChildren<ProjectActionsProps>) => {
  const menuItems = getProjectActionItems(projectId);

  return <TableActionPopover items={menuItems}>{children}</TableActionPopover>;
};

const getProjectActionItems = (projectId: string): MenuItem[] => {
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
    },
  ];
};
