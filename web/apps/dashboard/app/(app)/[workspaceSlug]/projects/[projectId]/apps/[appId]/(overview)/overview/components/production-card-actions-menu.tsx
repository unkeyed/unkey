"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { Deployment } from "@/lib/collections";
import { Ban, Bolt, Clone, Dots, Github, Hammer2 } from "@unkey/icons";
import { Button, toast } from "@unkey/ui";
import dynamic from "next/dynamic";
import { useMemo } from "react";
import { isRedeployableDeploymentStatus } from "../../deployments/components/table/components/actions/deployment-action-eligibility";
import type { DeploymentDisplayStatus } from "./status";

const RedeployDialog = dynamic(
  () =>
    import("../../deployments/components/table/components/actions/redeploy-dialog").then(
      (m) => m.RedeployDialog,
    ),
  { ssr: false },
);

type ProductionCardActionsMenuProps = {
  deployment: Deployment;
  status: DeploymentDisplayStatus;
  commitUrl?: string;
};

export function ProductionCardActionsMenu({
  deployment,
  status,
  commitUrl,
}: ProductionCardActionsMenuProps) {
  const items = useMemo((): MenuItem[] => {
    const stopped = status === "stopped";
    return [
      {
        id: "stop-wake",
        label: stopped ? "Wake" : "Stop",
        icon: stopped ? <Bolt iconSize="md-regular" /> : <Ban iconSize="md-regular" />,
        disabled: true,
        tooltip: "Available soon",
      },
      {
        id: "redeploy",
        label: "Redeploy",
        icon: <Hammer2 iconSize="md-regular" />,
        disabled: !isRedeployableDeploymentStatus(deployment.status),
        ActionComponent: (props) => <RedeployDialog {...props} selectedDeployment={deployment} />,
        divider: true,
      },
      {
        id: "copy-deployment-id",
        label: "Copy deployment ID",
        icon: <Clone iconSize="md-regular" />,
        onClick: () => {
          navigator.clipboard
            .writeText(deployment.id)
            .then(() => toast.success("Deployment ID copied to clipboard"))
            .catch(() => toast.error("Failed to copy to clipboard"));
        },
      },
      {
        id: "view-commit",
        label: "View commit on GitHub",
        icon: <Github iconSize="md-regular" />,
        disabled: !commitUrl,
        onClick: () => {
          if (commitUrl) {
            window.open(commitUrl, "_blank", "noopener,noreferrer");
          }
        },
      },
    ];
  }, [deployment, status, commitUrl]);

  return (
    <TableActionPopover items={items}>
      <Button
        variant="outline"
        size="sm"
        aria-label="More actions"
        className="w-7 p-0"
        onClick={(e) => e.stopPropagation()}
      >
        <Dots iconSize="sm-regular" />
      </Button>
    </TableActionPopover>
  );
}
