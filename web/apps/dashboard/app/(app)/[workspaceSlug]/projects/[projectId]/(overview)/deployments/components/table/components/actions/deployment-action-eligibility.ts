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

  // Rollback: only available for non-current, ready deployments in production
  const canRollback = isProduction && isReady && hasCurrent && !isCurrent;
  // Promote: same as rollback, but also allowed on the current deployment when rolled back
  // (to confirm the rollback and re-enable automatic deployments)
  const canPromote = isProduction && isReady && hasCurrent && (!isCurrent || ctx.isRolledBack);
  // Redeploy: available for any ready or failed deployment regardless of environment
  const canRedeploy = isReady || ctx.selectedDeployment.status === "failed";

  return { canRollback, canPromote, canRedeploy };
}
