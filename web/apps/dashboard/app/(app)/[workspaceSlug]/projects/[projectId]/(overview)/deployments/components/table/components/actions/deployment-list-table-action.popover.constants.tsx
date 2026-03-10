"use client";
import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/data-provider";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Deployment, Environment } from "@/lib/collections";
import { ArrowDottedRotateAnticlockwise, ChevronUp, Layers3 } from "@unkey/icons";
import { useRouter } from "next/navigation";
import { useMemo } from "react";
import { getDeploymentActionEligibility } from "./deployment-action-eligibility";
import { PromotionDialog } from "./promotion-dialog";
import { RedeployDialog } from "./redeploy-dialog";
import { RollbackDialog } from "./rollback-dialog";

type DeploymentListTableActionsProps = {
  currentDeployment?: Deployment;
  selectedDeployment: Deployment;
  environment?: Environment;
  isRolledBack: boolean;
};

export const DeploymentListTableActions = ({
  currentDeployment,
  selectedDeployment,
  environment,
  isRolledBack,
}: DeploymentListTableActionsProps) => {
  const workspace = useWorkspaceNavigation();
  const { getDomainsForDeployment } = useProjectData();
  const data = getDomainsForDeployment(selectedDeployment.id).map((domain) => ({
    host: domain.fullyQualifiedDomainName,
  }));

  const router = useRouter();
  // biome-ignore lint/correctness/useExhaustiveDependencies: its okay
  const menuItems = useMemo((): MenuItem[] => {

    const { canRollback, canPromote, canRedeploy } = getDeploymentActionEligibility({
      selectedDeployment,
      currentDeploymentId: currentDeployment?.id ?? null,
      isRolledBack,
      environmentSlug: environment?.slug ?? null,
    });

    return [
      {
        id: "rollback",
        label: "Rollback",
        icon: <ArrowDottedRotateAnticlockwise iconSize="md-regular" />,
        disabled: !canRollback,
        ActionComponent:
          currentDeployment && canRollback
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
        disabled: !canPromote,
        ActionComponent:
          canPromote
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
        ActionComponent: canRedeploy
          ? (props) => <RedeployDialog {...props} selectedDeployment={selectedDeployment} />
          : undefined,
      },
      {
        id: "sentinel-logs",
        label: "Go to requests...",
        icon: <Layers3 iconSize="md-regular" />,
        onClick: () => {
          router.push(
            `/${workspace.slug}/projects/${selectedDeployment.projectId}/requests?since=6h&deploymentId=contains:${selectedDeployment.id}`,
          );
        },
      },
      {
        id: "runtime-logs",
        label: "Go to logs...",
        icon: <Layers3 iconSize="md-regular" />,
        onClick: () => {
          router.push(`/${workspace.slug}/projects/${selectedDeployment.projectId}/logs`);
        },
      },
    ];
  }, [
    selectedDeployment.id,
    selectedDeployment.status,
    currentDeployment?.id,
    environment?.slug,
    isRolledBack,
    data,
  ]);

  return <TableActionPopover items={menuItems} />;
};
