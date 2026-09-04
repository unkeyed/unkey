"use client";
import { useAppCurrentDeployment } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/hooks/use-app-current-deployment";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import type { MenuItem } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Deployment, Environment } from "@/lib/collections";
import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import { routes } from "@/lib/navigation/routes";
import {
  ArrowDottedRotateAnticlockwise,
  ArrowOppositeDirectionY,
  Bolt,
  BoltSlash,
  ChevronUp,
  Hammer2,
  Layers3,
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
  const { currentDeployment, isRolledBack } = useAppCurrentDeployment();
  const { gated, openPaywall, planGate } = useDeployActionGate();

  const currentDeploymentId = currentDeployment?.id ?? null;
  const hasCurrentDeployment = currentDeployment !== undefined;

  // biome-ignore lint/correctness/useExhaustiveDependencies: matches the list menu
  const items = useMemo((): MenuItem[] => {
    const { canRollback, canPromote, canStop, canWake } = getDeploymentActionEligibility({
      selectedDeployment: { id: deployment.id, status },
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
        icon: <ArrowDottedRotateAnticlockwise iconSize="md-regular" />,
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
        icon: <ChevronUp iconSize="md-regular" />,
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
        icon: <Bolt iconSize="md-regular" />,
        disabled: !canWake,
        prefetch: async () => {
          await import("../components/table/components/actions/wake-dialog");
        },
        ActionComponent: (props) => <WakeDialog {...props} deployment={deployment} />,
      }),
      {
        id: "stop",
        label: "Stop deployment",
        icon: <BoltSlash iconSize="md-regular" />,
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
        icon: <Layers3 iconSize="md-regular" />,
        href: routes.projects.logs(deploymentScope),
      },
      {
        id: "request-logs",
        label: "Go to requests",
        icon: <ArrowOppositeDirectionY iconSize="md-regular" />,
        href: routes.projects.requests({ ...deploymentScope, since: "6h" }),
      },
      {
        id: "build-steps",
        label: "Go to build logs",
        icon: <Hammer2 iconSize="md-regular" />,
        href: routes.projects.apps.deployment({
          ...deploymentScope,
          appId: deployment.appId,
          build: true,
        }),
      },
    ];

    return [
      ...availableActions.map((action, index) =>
        index === availableActions.length - 1 ? { ...action, divider: true } : action,
      ),
      ...navigationActions,
    ];
  }, [
    deployment.id,
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
