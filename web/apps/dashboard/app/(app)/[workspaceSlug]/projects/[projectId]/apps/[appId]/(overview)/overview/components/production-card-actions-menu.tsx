"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { Deployment } from "@/lib/collections";
import {
  ArrowOppositeDirectionY,
  Ban,
  Bolt,
  Clone,
  Dots,
  Github,
  Hammer2,
  Layers3,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { Button, toast } from "@unkey/ui";
import type { Route } from "next";
import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
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
    const sourceItems = match(deployment.source)
      .returnType<MenuItem[]>()
      .with("git_build", () =>
        commitUrl
          ? [
              {
                id: "view-commit",
                label: "View commit on GitHub",
                icon: <Github iconSize="md-regular" />,
                onClick: () => window.open(commitUrl, "_blank", "noopener,noreferrer"),
              },
            ]
          : [],
      )
      .with("docker_image", "unknown", () => [])
      .exhaustive();
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
        icon: <Layers3 iconSize="md-regular" />,
        onClick: () => router.push(logsHref),
      },
      {
        id: "view-requests",
        label: "Go to requests",
        icon: <ArrowOppositeDirectionY iconSize="md-regular" />,
        onClick: () => router.push(requestsHref),
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
      ...sourceItems,
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
          <Dots iconSize="sm-regular" />
        </Button>
      </TableActionPopover>
      {planGate}
    </>
  );
}
