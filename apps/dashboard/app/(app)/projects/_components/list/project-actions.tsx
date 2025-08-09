"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { Project } from "@/lib/trpc/routers/deploy/project/list";
import { Clone, Gear, Layers3, Link4, Trash } from "@unkey/icons";

import { toast } from "@unkey/ui";
import type { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";
import { useRouter } from "next/navigation";
import type { PropsWithChildren } from "react";

type ProjectActionsProps = {
  project: Project;
};

export const ProjectActions = ({ project, children }: PropsWithChildren<ProjectActionsProps>) => {
  const router = useRouter();
  const menuItems = getProjectActionItems(project, router);

  return <TableActionPopover items={menuItems}>{children}</TableActionPopover>;
};

const getProjectActionItems = (project: Project, router: AppRouterInstance): MenuItem[] => {
  const primaryHostname = project.hostnames[0]?.hostname;

  return [
    {
      id: "favorite-project",
      label: "Add favorite",
      icon: <Gear size="md-regular" />,
      onClick: () => {},
      divider: true,
    },
    {
      id: "view-project",
      label: "View live API",
      icon: <Link4 size="md-regular" />,
      onClick: () => {
        if (primaryHostname) {
          window.open(`https://${primaryHostname}`, "_blank", "noopener,noreferrer");
        }
      },
      disabled: !primaryHostname,
    },
    {
      id: "copy-project-id",
      label: "Copy project ID",
      className: "mt-1",
      icon: <Clone size="md-regular" />,
      onClick: () => {
        navigator.clipboard
          .writeText(project.id)
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
      label: "View logs",
      icon: <Layers3 size="md-regular" />,
      onClick: () => {
        //INFO: This will change soon
        const fakeDeploymentId = "idk";
        router.push(`/projects/${project.id}/deployments/${fakeDeploymentId}/logs`);
      },
    },
    {
      id: "project-settings",
      label: "Project settings",
      icon: <Gear size="md-regular" />,
      onClick: () => {
        //INFO: This will change soon
        const fakeDeploymentId = "idk";
        router.push(`/projects/${project.id}/deployments/${fakeDeploymentId}/settings`);
      },
      divider: true,
    },
    {
      id: "delete-project",
      label: "Delete project",
      icon: <Trash size="md-regular" />,
      ActionComponent: () => null,
    },
  ];
};
