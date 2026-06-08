"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { ArrowOppositeDirectionY, Clone, Cloud, Gear, Layers3 } from "@unkey/icons";
import { toast } from "@unkey/ui";
import type { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";
import { useRouter } from "next/navigation";
import type { PropsWithChildren } from "react";

type AppActionsProps = {
  projectId: string;
  appId: string;
};

export const AppActions = ({ projectId, appId, children }: PropsWithChildren<AppActionsProps>) => {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const menuItems = getAppActionItems(workspace.slug, projectId, appId, router);

  return <TableActionPopover items={menuItems}>{children}</TableActionPopover>;
};

const getAppActionItems = (
  workspaceSlug: string,
  projectId: string,
  appId: string,
  router: AppRouterInstance,
): MenuItem[] => {
  const appBase = `/${workspaceSlug}/projects/${projectId}/apps/${appId}`;

  return [
    {
      id: "copy-app-id",
      label: "Copy app ID",
      icon: <Clone iconSize="md-medium" />,
      onClick: () => {
        navigator.clipboard
          .writeText(appId)
          .then(() => {
            toast.success("App ID copied to clipboard");
          })
          .catch((error) => {
            console.error("Failed to copy to clipboard:", error);
            toast.error("Failed to copy to clipboard");
          });
      },
      divider: true,
    },
    {
      id: "view-logs",
      label: "View logs",
      icon: <Layers3 iconSize="md-regular" />,
      onClick: () => {
        router.push(`/${workspaceSlug}/projects/${projectId}/logs?appId=${appId}`);
      },
    },
    {
      id: "view-requests",
      label: "View requests",
      icon: <ArrowOppositeDirectionY iconSize="md-regular" />,
      onClick: () => {
        router.push(`/${workspaceSlug}/projects/${projectId}/requests?since=6h&appId=${appId}`);
      },
    },
    {
      id: "view-deployments",
      label: "View deployments",
      icon: <Cloud iconSize="md-regular" />,
      onClick: () => {
        router.push(`${appBase}/deployments`);
      },
    },
    {
      id: "app-settings",
      label: "App settings",
      icon: <Gear iconSize="md-medium" />,
      onClick: () => {
        router.push(`${appBase}/settings`);
      },
    },
  ];
};
