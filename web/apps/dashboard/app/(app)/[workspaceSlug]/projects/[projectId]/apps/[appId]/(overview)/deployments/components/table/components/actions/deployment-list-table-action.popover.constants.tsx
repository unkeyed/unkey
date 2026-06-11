"use client";
import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Deployment, Environment } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import {
  ArrowDottedRotateAnticlockwise,
  ArrowOppositeDirectionY,
  Ban,
  ChevronUp,
  Hammer2,
  Layers3,
} from "@unkey/icons";
import { useRouter } from "next/navigation";
import { useMemo } from "react";
import { CancelDialog } from "./cancel-dialog";
import { getDeploymentActionEligibility } from "./deployment-action-eligibility";
import { PromotionDialog } from "./promotion-dialog";
import { RedeployDialog } from "./redeploy-dialog";
import { RollbackDialog } from "./rollback-dialog";

type DeploymentListTableActionsProps = {
  selectedDeployment: Deployment;
  environment?: Environment;
};

export const DeploymentListTableActions = ({
  selectedDeployment,
  environment,
}: DeploymentListTableActionsProps) => {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const { getDeploymentById, project } = useProjectData();

  const currentDeploymentId = project?.currentDeploymentId ?? null;
  const isRolledBack = Boolean(project?.isRolledBack);
  const currentDeployment = getDeploymentById(currentDeploymentId ?? "");
  const hasCurrentDeployment = currentDeployment !== undefined;

  // biome-ignore lint/correctness/useExhaustiveDependencies: its okay
  const menuItems = useMemo((): MenuItem[] => {
    const { canRollback, canPromote, canRedeploy, canCancel } = getDeploymentActionEligibility({
      selectedDeployment,
      currentDeploymentId,
      isRolledBack,
      environmentSlug: environment?.slug ?? null,
    });

    return [
      {
        id: "rollback",
        label: "Rollback",
        icon: <ArrowDottedRotateAnticlockwise iconSize="md-regular" />,
        disabled: !canRollback || !hasCurrentDeployment,
        ActionComponent: hasCurrentDeployment
          ? (props) => (
              <RollbackDialog
                {...props}
                currentDeployment={currentDeployment}
                targetDeployment={selectedDeployment}
              />
            )
          : undefined,
      },
      {
        id: "Promote",
        label: "Promote",
        icon: <ChevronUp iconSize="md-regular" />,
        disabled: !canPromote || !hasCurrentDeployment,
        ActionComponent: hasCurrentDeployment
          ? (props) => (
              <PromotionDialog
                {...props}
                currentDeployment={currentDeployment}
                targetDeployment={selectedDeployment}
              />
            )
          : undefined,
      },
      {
        id: "redeploy",
        label: "Redeploy",
        icon: <ArrowDottedRotateAnticlockwise iconSize="md-regular" />,
        disabled: !canRedeploy,
        ActionComponent: (props) => (
          <RedeployDialog {...props} selectedDeployment={selectedDeployment} />
        ),
      },
      {
        id: "cancel",
        label: "Cancel deployment",
        icon: <Ban iconSize="md-regular" />,
        disabled: !canCancel,
        ActionComponent: (props) => <CancelDialog {...props} deployment={selectedDeployment} />,
      },
      {
        id: "sentinel-logs",
        label: "Go to requests",
        icon: <ArrowOppositeDirectionY iconSize="md-regular" />,
        onClick: () => {
          router.push(
            routes.projects.requests({
              workspaceSlug: workspace.slug,
              projectId: selectedDeployment.projectId,
              since: "6h",
              deploymentId: selectedDeployment.id,
            }),
          );
        },
      },
      {
        id: "runtime-logs",
        label: "Go to logs",
        icon: <Layers3 iconSize="md-regular" />,
        onClick: () => {
          router.push(
            routes.projects.logs({
              workspaceSlug: workspace.slug,
              projectId: selectedDeployment.projectId,
              appId: selectedDeployment.appId,
              deploymentId: selectedDeployment.id,
            }),
          );
        },
      },
      {
        id: "build-steps",
        label: "Go to build logs",
        icon: <Hammer2 iconSize="md-regular" />,
        onClick: () => {
          router.push(
            routes.projects.apps.deployment({
              workspaceSlug: workspace.slug,
              projectId: selectedDeployment.projectId,
              appId: selectedDeployment.appId,
              deploymentId: selectedDeployment.id,
              build: true,
            }),
          );
        },
      },
    ];
  }, [
    selectedDeployment.id,
    selectedDeployment.status,
    currentDeploymentId,
    isRolledBack,
    environment?.slug,
    hasCurrentDeployment,
  ]);

  return <TableActionPopover items={menuItems} />;
};
