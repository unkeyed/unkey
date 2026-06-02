"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Clone, Cloud, Gear, Layers3 } from "@unkey/icons";
import { toast } from "@unkey/ui";
import type { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";
import { useRouter } from "next/navigation";
import type { PropsWithChildren } from "react";

type AppActionsProps = {
  projectSlug: string;
  appSlug: string;
  appId: string;
};

export const AppActions = ({
  projectSlug,
  appSlug,
  appId,
  children,
}: PropsWithChildren<AppActionsProps>) => {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const menuItems = getAppActionItems(workspace.slug, projectSlug, appSlug, appId, router);

  return <TableActionPopover items={menuItems}>{children}</TableActionPopover>;
};

const getAppActionItems = (
  workspaceSlug: string,
  projectSlug: string,
  appSlug: string,
  appId: string,
  router: AppRouterInstance,
): MenuItem[] => {
  const appBase = `/${workspaceSlug}/projects/${projectSlug}/apps/${appSlug}`;

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
      id: "view-requests",
      label: "View requests",
      icon: <Layers3 iconSize="md-regular" />,
      onClick: () => {
        router.push(`${appBase}/requests`);
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
