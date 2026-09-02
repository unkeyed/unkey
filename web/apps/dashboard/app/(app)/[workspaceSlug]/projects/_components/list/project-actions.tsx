"use client";

import { toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import {
  IconArrowsOppositeDirectionYOutline18,
  IconCloneOutline18,
  IconGearOutline18,
  IconLayers3Outline18,
} from "nucleo-ui-outline-18";
import type { PropsWithChildren } from "react";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";

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
      icon: <IconCloneOutline18 className="size-4" />,
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
      icon: <IconArrowsOppositeDirectionYOutline18 className="size-4" />,
      onClick: () => {
        router.push(routes.projects.requests({ workspaceSlug: workspace.slug, projectId }));
      },
    },
    {
      id: "view-logs",
      label: "View logs",
      icon: <IconLayers3Outline18 className="size-4" />,
      onClick: () => {
        router.push(routes.projects.logs({ workspaceSlug: workspace.slug, projectId }));
      },
    },
    {
      id: "project-settings",
      label: "Project settings",
      icon: <IconGearOutline18 className="size-4" />,
      onClick: () => {
        router.push(routes.projects.settings({ workspaceSlug: workspace.slug, projectId }));
      },
    },
  ];

  return <TableActionPopover items={menuItems}>{children}</TableActionPopover>;
};
