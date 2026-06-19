"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { Deployment } from "@/lib/collections";
import { Ban, Bolt, Clone, Github, Hammer2 } from "@unkey/icons";
import { toast } from "@unkey/ui";
import dynamic from "next/dynamic";
import { useMemo } from "react";
import { isRedeployableDeploymentStatus } from "../../deployments/components/table/components/actions/deployment-action-eligibility";
import type { DeploymentDisplayStatus } from "./production-deployment-card-view";

const RedeployDialog = dynamic(
  () =>
    import("../../deployments/components/table/components/actions/redeploy-dialog").then(
      (m) => m.RedeployDialog,
    ),
  { ssr: false },
);

type ProductionCardActionsMenuProps = {
  deployment: Deployment;
  // Display status drives the Stop/Wake label. Eligibility for the real
  // mutations keys off deployment.status (the canonical value), not this.
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
        // TODO: wire to the start/stop mutation from Andreas's PR. Until it
        // lands there is no capability to call, so the item is inert. Add an
        // ActionComponent with a confirm dialog (mirror RedeployDialog) then.
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

  return <TableActionPopover items={items} />;
}
