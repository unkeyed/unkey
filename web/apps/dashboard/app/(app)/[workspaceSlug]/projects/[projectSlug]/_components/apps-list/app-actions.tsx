"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { ArrowOppositeDirectionY, Clone, Cloud, Gear, Layers3 } from "@unkey/icons";
import { toast } from "@unkey/ui";
import type { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";
import { useRouter } from "next/navigation";
import type { PropsWithChildren } from "react";

type AppActionsProps = {
  basePath: string;
  appSlug: string;
  appId: string;
};

export const AppActions = ({
  basePath,
  appSlug,
  appId,
  children,
}: PropsWithChildren<AppActionsProps>) => {
  const router = useRouter();
  const menuItems = getAppActionItems(basePath, appSlug, appId, router);

  return <TableActionPopover items={menuItems}>{children}</TableActionPopover>;
};

const getAppActionItems = (
  basePath: string,
  appSlug: string,
  appId: string,
  router: AppRouterInstance,
): MenuItem[] => {
  const appBase = `${basePath}/apps/${appSlug}`;

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
        router.push(`${basePath}/logs?appId=${appId}`);
      },
    },
    {
      id: "view-requests",
      label: "View requests",
      icon: <ArrowOppositeDirectionY iconSize="md-regular" />,
      onClick: () => {
        router.push(`${basePath}/requests?since=6h&appId=${appId}`);
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
