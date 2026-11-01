"use client";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import type { MenuItem } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import {
  type Deployment,
  type Environment,
  hasBuildLogs,
  useApp,
  useCurrentDeployment,
} from "@/lib/deploy-data";
import { useFlag } from "@/lib/flags/provider";
import { routes } from "@/lib/navigation/routes";
import {
  IconArrowDottedRotateAnticlockwiseOutline18,
  IconArrowsOppositeDirectionYOutline18,
  IconBoltOutline18,
  IconBoltSlashOutline18,
  IconChevronUpOutline18,
  IconHammer2Outline18,
  IconLayers3Outline18,
} from "@unkey/icons";
import dynamic from "next/dynamic";
import { useMemo } from "react";
import { getDeploymentActionEligibility } from "../components/table/components/actions/deployment-action-eligibility";

const PromotionDialog = dynamic(
  () =>
    import("../components/table/components/actions/promotion-dialog").then(
      (m) => m.PromotionDialog,
    ),
  { ssr: false },
);
const RollbackDialog = dynamic(
  () =>
    import("../components/table/components/actions/rollback-dialog").then((m) => m.RollbackDialog),
  { ssr: false },
);
const StopDialog = dynamic(
  () => import("../components/table/components/actions/stop-dialog").then((m) => m.StopDialog),
  {
    ssr: false,
  },
);
const WakeDialog = dynamic(
  () => import("../components/table/components/actions/wake-dialog").then((m) => m.WakeDialog),
  {
    ssr: false,
  },
);

type UseDeploymentHeaderActionsProps = {
  deployment: Deployment;
  environment?: Environment;
  status: DeploymentStatus;
};

type DeploymentHeaderActions = {
  items: MenuItem[];
  planGate: React.ReactNode;
  gated: boolean;
  openPaywall: () => void;
};

const isItemDisabled = (item: MenuItem): boolean =>
  typeof item.disabled === "function" ? item.disabled() : Boolean(item.disabled);

export function useDeploymentHeaderActions({
  deployment,
  environment,
  status,
}: UseDeploymentHeaderActionsProps): DeploymentHeaderActions {
  const workspace = useWorkspaceNavigation();
  const { data: app } = useApp();
  const currentDeployment = useCurrentDeployment().data ?? undefined;
  const isRolledBack = app?.isRolledBack ?? false;
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const appV2 = useFlag("appV2");

  const currentDeploymentId = currentDeployment?.id ?? null;
  const hasCurrentDeployment = currentDeployment !== undefined;

  // biome-ignore lint/correctness/useExhaustiveDependencies: matches the list menu
  const items = useMemo((): MenuItem[] => {
    const { canRollback, canPromote, canStop, canWake } = getDeploymentActionEligibility({
      selectedDeployment: { id: deployment.id, status, desiredState: deployment.desiredState },
      currentDeploymentId,
      isRolledBack,
      environmentKind: environment?.kind ?? null,
    });

    const gateAction = (item: MenuItem): MenuItem =>
      gated && !isItemDisabled(item)
        ? { ...item, ActionComponent: undefined, onClick: () => openPaywall() }
        : item;

    const deploymentScope = {
      workspaceSlug: workspace.slug,
      projectId: deployment.projectId,
      deploymentId: deployment.id,
    };

    const stateActions: MenuItem[] = [
      gateAction({
        id: "rollback",
        label: "Rollback",
        icon: <IconArrowDottedRotateAnticlockwiseOutline18 className="size-3.5" />,
        disabled: !canRollback || !hasCurrentDeployment,
        prefetch: async () => {
          await import("../components/table/components/actions/rollback-dialog");
        },
        ActionComponent: hasCurrentDeployment
          ? (props) => (
              <RollbackDialog
                {...props}
                currentDeployment={currentDeployment}
                targetDeployment={deployment}
              />
            )
          : undefined,
      }),
      gateAction({
        id: "promote",
        label: "Promote",
        icon: <IconChevronUpOutline18 className="size-3.5" />,
        disabled: !canPromote || !hasCurrentDeployment,
        prefetch: async () => {
          await import("../components/table/components/actions/promotion-dialog");
        },
        ActionComponent: hasCurrentDeployment
          ? (props) => (
              <PromotionDialog
                {...props}
                currentDeployment={currentDeployment}
                targetDeployment={deployment}
              />
            )
          : undefined,
      }),
      gateAction({
        id: "wake",
        label: "Wake deployment",
        icon: <IconBoltOutline18 className="size-3.5" />,
        disabled: !canWake,
        prefetch: async () => {
          await import("../components/table/components/actions/wake-dialog");
        },
        ActionComponent: (props) => <WakeDialog {...props} deployment={deployment} />,
      }),
      {
        id: "stop",
        label: "Stop deployment",
        icon: <IconBoltSlashOutline18 className="size-3.5" />,
        disabled: !canStop,
        prefetch: async () => {
          await import("../components/table/components/actions/stop-dialog");
        },
        ActionComponent: (props) => <StopDialog {...props} deployment={deployment} />,
      },
    ];
    const availableActions = stateActions.filter((action) => !isItemDisabled(action));

    const navigationActions: MenuItem[] = [
      {
        id: "runtime-logs",
        label: "Go to logs",
        icon: <IconLayers3Outline18 className="size-3.5" />,
        href: routes.projects.logs(deploymentScope),
      },
      {
        id: "request-logs",
        label: "Go to requests",
        icon: <IconArrowsOppositeDirectionYOutline18 className="size-3.5" />,
        href: routes.projects.requests({ ...deploymentScope, since: "6h" }),
      },
    ];
    if (!appV2 || hasBuildLogs({ source: deployment.source, status })) {
      navigationActions.push({
        id: "build-steps",
        label: "Go to build logs",
        icon: <IconHammer2Outline18 className="size-3.5" />,
        href: routes.projects.apps.deployment({
          ...deploymentScope,
          appId: deployment.appId,
          build: true,
        }),
        replace: appV2,
      });
    }

    return [
      ...availableActions.map((action, index) =>
        index === availableActions.length - 1 ? { ...action, divider: true } : action,
      ),
      ...navigationActions,
    ];
  }, [
    deployment.id,
    deployment.desiredState,
    deployment.source,
    appV2,
    status,
    currentDeploymentId,
    isRolledBack,
    environment?.kind,
    hasCurrentDeployment,
    gated,
    openPaywall,
  ]);

  return { items, planGate, gated, openPaywall };
}
