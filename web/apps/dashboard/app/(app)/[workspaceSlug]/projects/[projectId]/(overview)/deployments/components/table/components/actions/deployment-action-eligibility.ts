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
  const status = ctx.selectedDeployment.status;
  const isActionable = status === "ready";
  const isProduction = ctx.environmentSlug === "production";
  const hasCurrent = ctx.currentDeploymentId !== null;
  const isCurrent = hasCurrent && ctx.currentDeploymentId === ctx.selectedDeployment.id;

  // Rollback: available for non-current, ready deployments in production
  const canRollback = isProduction && isActionable && hasCurrent && !isCurrent;
  // Promote: same as rollback, but also allowed on the current deployment when rolled back.
  const canPromote = isProduction && isActionable && hasCurrent && (!isCurrent || ctx.isRolledBack);
  // Redeploy: available for ready, idle, or failed deployments regardless of environment
  const canRedeploy = isActionable || status === "stopped" || status === "failed";

  return { canRollback, canPromote, canRedeploy };
}
