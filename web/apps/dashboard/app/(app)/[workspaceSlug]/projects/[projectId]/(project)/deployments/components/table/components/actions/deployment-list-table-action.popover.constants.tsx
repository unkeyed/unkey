"use client";
import { useProject } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/layout-provider";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Deployment, Environment } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { ArrowDottedRotateAnticlockwise, ChevronUp, Layers3 } from "@unkey/icons";
import { useRouter } from "next/navigation";
import { useMemo } from "react";
import { PromotionDialog } from "./promotion-dialog";
import { RollbackDialog } from "./rollback-dialog";

type DeploymentListTableActionsProps = {
  liveDeployment?: Deployment;
  selectedDeployment: Deployment;
  environment?: Environment;
};

export const DeploymentListTableActions = ({
  liveDeployment,
  selectedDeployment,
  environment,
}: DeploymentListTableActionsProps) => {
  const workspace = useWorkspaceNavigation();
  const { collections } = useProject();
  const { data } = useLiveQuery((q) =>
    q
      .from({ domain: collections.domains })
      .where(({ domain }) => eq(domain.deploymentId, selectedDeployment.id))
      .select(({ domain }) => ({ host: domain.fullyQualifiedDomainName })),
  );

  const router = useRouter();
  // biome-ignore lint/correctness/useExhaustiveDependencies: its okay
  const menuItems = useMemo((): MenuItem[] => {
    const canRollbackAndRollback =
      liveDeployment &&
      environment?.slug === "production" &&
      selectedDeployment.status === "ready" &&
      selectedDeployment.id !== liveDeployment.id;

    return [
      {
        id: "rollback",
        label: "Rollback",
        icon: <ArrowDottedRotateAnticlockwise iconSize="md-regular" />,
        disabled: !canRollbackAndRollback,
        ActionComponent:
          liveDeployment && canRollbackAndRollback
            ? (props) => (
                <RollbackDialog
                  {...props}
                  liveDeployment={liveDeployment}
                  targetDeployment={selectedDeployment}
                />
              )
            : undefined,
      },
      {
        id: "Promote",
        label: "Promote",
        icon: <ChevronUp iconSize="md-regular" />,
        disabled: !canRollbackAndRollback,
        ActionComponent:
          liveDeployment && canRollbackAndRollback
            ? (props) => (
                <PromotionDialog
                  {...props}
                  liveDeployment={liveDeployment}
                  targetDeployment={selectedDeployment}
                />
              )
            : undefined,
      },

      {
        id: "sentinel-logs",
        label: "Go to Sentinel Logs...",
        icon: <Layers3 iconSize="md-regular" />,
        onClick: () => {
          //INFO: This will produce a long query, but once we start using `contains` instead of `is` this will be a shorter query.
          router.push(
            `/${workspace.slug}/projects/${selectedDeployment.projectId}/sentinel-logs?host=${data
              .map((item) => `is:${item.host}`)
              .join(",")}`,
          );
        },
      },
    ];
  }, [
    selectedDeployment.id,
    selectedDeployment.status,
    liveDeployment?.id,
    environment?.slug,
    data,
  ]);

  return <TableActionPopover items={menuItems} />;
};
