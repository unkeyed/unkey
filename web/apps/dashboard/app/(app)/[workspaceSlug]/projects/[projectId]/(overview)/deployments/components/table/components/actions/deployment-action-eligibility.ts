import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";

type DeploymentActionContext = {
  selectedDeployment: { id: string; status: DeploymentStatus };
  currentDeploymentId: string | null;
  isRolledBack: boolean;
  environmentSlug: string | null;
};

type DeploymentActionEligibility = {
  canRollback: boolean;
  canPromote: boolean;
  canRedeploy: boolean;
};

export function getDeploymentActionEligibility(
  ctx: DeploymentActionContext,
): DeploymentActionEligibility {
  const isReady = ctx.selectedDeployment.status === "ready";
  const isProduction = ctx.environmentSlug === "production";
  const hasCurrent = ctx.currentDeploymentId !== null;
  const isCurrent = hasCurrent && ctx.currentDeploymentId === ctx.selectedDeployment.id;

  const canRollback = isProduction && isReady && hasCurrent && !isCurrent;
  const canPromote = isProduction && isReady && !isCurrent && ctx.isRolledBack;
  const canRedeploy = isReady || ctx.selectedDeployment.status === "failed";

  return { canRollback, canPromote, canRedeploy };
}
