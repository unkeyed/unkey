"use client";
import { useProjectLayout } from "@/app/(app)/projects/[projectId]/layout-provider";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { Deployment, Environment } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { ArrowDottedRotateAnticlockwise, Layers3 } from "@unkey/icons";
import { useRouter } from "next/navigation";
import { useMemo } from "react";
import { RollbackDialog } from "../../../rollback-dialog";

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
  const { collections } = useProjectLayout();
  const { data } = useLiveQuery((q) =>
    q
      .from({ domain: collections.domains })
      .where(({ domain }) => eq(domain.deploymentId, selectedDeployment.id))
      .select(({ domain }) => ({ host: domain.domain })),
  );

  const router = useRouter();
  // biome-ignore lint/correctness/useExhaustiveDependencies: its okay
  const menuItems = useMemo((): MenuItem[] => {
    // Rollback is only enabled when:
    // Selected deployment is not the current live deployment
    // Selected deployment status is "ready"
    // Environment is production, when testing locally if you don't use `--prod=env` flag rollback won't work.
    const isCurrentlyLive = liveDeployment?.id === selectedDeployment.id;
    const isDeploymentReady = selectedDeployment.status === "ready";
    const isProductionEnv = environment?.slug === "production";

    const canRollback = !isCurrentlyLive && isDeploymentReady && isProductionEnv;

    return [
      {
        id: "rollback",
        label: "Rollback",
        icon: <ArrowDottedRotateAnticlockwise size="md-regular" />,
        disabled: !canRollback,
        ActionComponent:
          liveDeployment && canRollback
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
        id: "gateway-logs",
        label: "Go to Gateway Logs...",
        icon: <Layers3 size="md-regular" />,
        onClick: () => {
          //INFO: This will produce a long query, but once we start using `contains` instead of `is` this will be a shorter query.
          router.push(
            `/projects/${selectedDeployment.projectId}/gateway-logs?host=${data
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
