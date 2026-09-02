"use client";

import { Github } from "@unkey/icons";
import { Button, toast } from "@unkey/ui";
import type { Route } from "next";
import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
import {
  IconArrowsOppositeDirectionYOutline18,
  IconBanOutline18,
  IconBoltOutline18,
  IconCloneOutline18,
  IconDotsOutline18,
  IconHammer2Outline18,
  IconLayers3Outline18,
} from "nucleo-ui-outline-18";
import { useMemo } from "react";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { Deployment } from "@/lib/collections";
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
  logsHref: Route;
  requestsHref: Route;
};

export function ProductionCardActionsMenu({
  deployment,
  status,
  commitUrl,
  logsHref,
  requestsHref,
}: ProductionCardActionsMenuProps) {
  const router = useRouter();
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const items = useMemo((): MenuItem[] => {
    const stopped = status === "stopped";
    const canRedeploy = isRedeployableDeploymentStatus(deployment.status);
    return [
      {
        id: "stop-wake",
        label: stopped ? "Wake" : "Stop",
        icon: stopped ? (
          <IconBoltOutline18 className="size-4" />
        ) : (
          <IconBanOutline18 className="size-4" />
        ),
        disabled: true,
        tooltip: "Available soon",
      },
      {
        id: "redeploy",
        label: "Redeploy",
        icon: <IconHammer2Outline18 className="size-4" />,
        disabled: !canRedeploy,
        // Without a Compute plan, redeploy opens the paywall instead of building.
        ...(gated && canRedeploy
          ? { onClick: () => openPaywall() }
          : {
              ActionComponent: (props) => (
                <RedeployDialog {...props} selectedDeployment={deployment} />
              ),
            }),
        divider: true,
      },
      {
        id: "view-logs",
        label: "Go to logs",
        icon: <IconLayers3Outline18 className="size-4" />,
        onClick: () => router.push(logsHref),
      },
      {
        id: "view-requests",
        label: "Go to requests",
        icon: <IconArrowsOppositeDirectionYOutline18 className="size-4" />,
        onClick: () => router.push(requestsHref),
        divider: true,
      },
      {
        id: "copy-deployment-id",
        label: "Copy deployment ID",
        icon: <IconCloneOutline18 className="size-4" />,
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
        icon: <Github className="size-4" />,
        disabled: !commitUrl,
        onClick: () => {
          if (commitUrl) {
            window.open(commitUrl, "_blank", "noopener,noreferrer");
          }
        },
      },
    ];
  }, [deployment, status, commitUrl, gated, openPaywall, router, logsHref, requestsHref]);

  return (
    <>
      <TableActionPopover items={items}>
        <Button
          variant="outline"
          size="sm"
          aria-label="More actions"
          className="w-7 p-0"
          onClick={(e) => e.stopPropagation()}
        >
          <IconDotsOutline18 />
        </Button>
      </TableActionPopover>
      {planGate}
    </>
  );
}
