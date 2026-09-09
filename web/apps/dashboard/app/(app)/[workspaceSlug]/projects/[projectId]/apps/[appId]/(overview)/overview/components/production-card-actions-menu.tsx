"use client";

import { Github } from "@unkey/icons";
import { match } from "@unkey/match";
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
  IconLayers2Outline18,
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
  deploymentHref: Route;
  logsHref: Route;
  requestsHref: Route;
};

export function ProductionCardActionsMenu({
  deployment,
  status,
  commitUrl,
  deploymentHref,
  logsHref,
  requestsHref,
}: ProductionCardActionsMenuProps) {
  const router = useRouter();
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const items = useMemo((): MenuItem[] => {
    const stopped = status === "stopped";
    const canRedeploy = isRedeployableDeploymentStatus(deployment.status);
    const sourceItems = match(deployment.source)
      .returnType<MenuItem[]>()
      .with("git", () =>
        commitUrl
          ? [
              {
                id: "view-commit",
                label: "View commit on GitHub",
                icon: <Github />,
                onClick: () => window.open(commitUrl, "_blank", "noopener,noreferrer"),
              },
            ]
          : [],
      )
      .with("oci", "unknown", () => [])
      .exhaustive();
    return [
      {
        id: "stop-wake",
        label: stopped ? "Wake" : "Stop",
        icon: stopped ? <IconBoltOutline18 /> : <IconBanOutline18 />,
        disabled: true,
        tooltip: "Available soon",
      },
      {
        id: "redeploy",
        label: "Redeploy",
        icon: <IconHammer2Outline18 />,
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
        id: "view-deployment",
        label: "Go to deployment",
        icon: <IconLayers2Outline18 />,
        href: deploymentHref,
      },
      {
        id: "view-logs",
        label: "Go to logs",
        icon: <IconLayers3Outline18 />,
        onClick: () => router.push(logsHref),
      },
      {
        id: "view-requests",
        label: "Go to requests",
        icon: <IconArrowsOppositeDirectionYOutline18 />,
        onClick: () => router.push(requestsHref),
        divider: true,
      },
      {
        id: "copy-deployment-id",
        label: "Copy deployment ID",
        icon: <IconCloneOutline18 />,
        onClick: () => {
          navigator.clipboard
            .writeText(deployment.id)
            .then(() => toast.success("Deployment ID copied to clipboard"))
            .catch(() => toast.error("Failed to copy to clipboard"));
        },
      },
      ...sourceItems,
    ];
  }, [
    deployment,
    status,
    commitUrl,
    gated,
    openPaywall,
    router,
    deploymentHref,
    logsHref,
    requestsHref,
  ]);

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
