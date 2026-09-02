"use client";
import { useRouter } from "next/navigation";
import {
  IconArrowDottedRotateAnticlockwiseOutline18,
  IconArrowsOppositeDirectionYOutline18,
  IconBanOutline18,
  IconBoltOutline18,
  IconBoltSlashOutline18,
  IconChevronUpOutline18,
  IconHammer2Outline18,
  IconLayers3Outline18,
} from "nucleo-ui-outline-18";
import { useMemo } from "react";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Deployment, Environment } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import { CancelDialog } from "./cancel-dialog";
import { getDeploymentActionEligibility } from "./deployment-action-eligibility";
import { PromotionDialog } from "./promotion-dialog";
import { RedeployDialog } from "./redeploy-dialog";
import { RollbackDialog } from "./rollback-dialog";
import { StopDialog } from "./stop-dialog";
import { WakeDialog } from "./wake-dialog";

type DeploymentListTableActionsProps = {
  selectedDeployment: Deployment;
  environment?: Environment;
};

const isItemDisabled = (disabled: MenuItem["disabled"]): boolean =>
  typeof disabled === "function" ? disabled() : Boolean(disabled);

export const DeploymentListTableActions = ({
  selectedDeployment,
  environment,
}: DeploymentListTableActionsProps) => {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const { getDeploymentById, project } = useProjectData();
  const { gated, openPaywall, planGate } = useDeployActionGate();

  const currentDeploymentId = project?.currentDeploymentId ?? null;
  const isRolledBack = Boolean(project?.isRolledBack);
  const currentDeployment = getDeploymentById(currentDeploymentId ?? "");
  const hasCurrentDeployment = currentDeployment !== undefined;

  // biome-ignore lint/correctness/useExhaustiveDependencies: its okay
  const menuItems = useMemo((): MenuItem[] => {
    const { canRollback, canPromote, canRedeploy, canCancel, canStop, canWake } =
      getDeploymentActionEligibility({
        selectedDeployment,
        currentDeploymentId,
        isRolledBack,
        environmentKind: environment?.kind ?? null,
      });

    // Without a Compute plan, actions that build or activate compute open the
    // paywall instead of their dialog. Cancel/stop de-escalate, so stay usable.
    const gateAction = (item: MenuItem): MenuItem =>
      gated && !isItemDisabled(item.disabled)
        ? { ...item, ActionComponent: undefined, onClick: () => openPaywall() }
        : item;

    return [
      gateAction({
        id: "rollback",
        label: "Rollback",
        icon: <IconArrowDottedRotateAnticlockwiseOutline18 className="size-4" />,
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
      }),
      gateAction({
        id: "Promote",
        label: "Promote",
        icon: <IconChevronUpOutline18 className="size-4" />,
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
      }),
      gateAction({
        id: "wake",
        label: "Wake deployment",
        icon: <IconBoltOutline18 className="size-4" />,
        disabled: !canWake,
        ActionComponent: (props) => <WakeDialog {...props} deployment={selectedDeployment} />,
      }),
      {
        id: "stop",
        label: "Stop deployment",
        icon: <IconBoltSlashOutline18 className="size-4" />,
        disabled: !canStop,
        ActionComponent: (props) => <StopDialog {...props} deployment={selectedDeployment} />,
      },
      gateAction({
        id: "redeploy",
        label: "Redeploy",
        icon: <IconArrowDottedRotateAnticlockwiseOutline18 className="size-4" />,
        disabled: !canRedeploy,
        ActionComponent: (props) => (
          <RedeployDialog {...props} selectedDeployment={selectedDeployment} />
        ),
      }),
      {
        id: "cancel",
        label: "Cancel deployment",
        icon: <IconBanOutline18 className="size-4" />,
        disabled: !canCancel,
        ActionComponent: (props) => <CancelDialog {...props} deployment={selectedDeployment} />,
      },
      {
        id: "request-logs",
        label: "Go to requests",
        icon: <IconArrowsOppositeDirectionYOutline18 className="size-4" />,
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
        icon: <IconLayers3Outline18 className="size-4" />,
        onClick: () => {
          router.push(
            routes.projects.logs({
              workspaceSlug: workspace.slug,
              projectId: selectedDeployment.projectId,
              deploymentId: selectedDeployment.id,
            }),
          );
        },
      },
      {
        id: "build-steps",
        label: "Go to build logs",
        icon: <IconHammer2Outline18 className="size-4" />,
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
    gated,
    openPaywall,
  ]);

  return (
    <>
      <TableActionPopover items={menuItems} />
      {planGate}
    </>
  );
};
