"use client";

import { toast } from "@unkey/ui";
import type { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";
import { useRouter } from "next/navigation";
import {
  IconArrowsOppositeDirectionYOutline18,
  IconCloneOutline18,
  IconCloudOutline18,
  IconGearOutline18,
  IconLayers3Outline18,
} from "nucleo-ui-outline-18";
import type { PropsWithChildren } from "react";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";

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
  const appScope = { workspaceSlug, projectId, appId };

  return [
    {
      id: "copy-app-id",
      label: "Copy app ID",
      icon: <IconCloneOutline18 className="size-4" />,
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
      icon: <IconLayers3Outline18 className="size-4" />,
      onClick: () => {
        router.push(routes.projects.logs(appScope));
      },
    },
    {
      id: "view-requests",
      label: "View requests",
      icon: <IconArrowsOppositeDirectionYOutline18 className="size-4" />,
      onClick: () => {
        router.push(routes.projects.requests({ ...appScope, since: "6h" }));
      },
    },
    {
      id: "view-deployments",
      label: "View deployments",
      icon: <IconCloudOutline18 className="size-4" />,
      onClick: () => {
        router.push(routes.projects.apps.deployments(appScope));
      },
    },
    {
      id: "app-settings",
      label: "App settings",
      icon: <IconGearOutline18 className="size-4" />,
      onClick: () => {
        router.push(routes.projects.apps.settings(appScope));
      },
    },
  ];
};
