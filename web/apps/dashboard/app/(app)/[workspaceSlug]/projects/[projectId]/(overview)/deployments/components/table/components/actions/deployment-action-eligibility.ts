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
  canCancel: boolean;
};

// Non-terminal statuses where a cancel is meaningful: the Restate workflow
// is still running, so we can abort it. Terminal statuses (ready, failed,
// skipped, stopped) are rejected by the backend anyway.
const CANCELLABLE_STATUSES: DeploymentStatus[] = [
  "pending",
  "starting",
  "building",
  "deploying",
  "network",
  "finalizing",
  "awaiting_approval",
];

// isCancellableDeploymentStatus reports whether a deployment is still in
// flight and can therefore be cancelled. Exported so UI elements outside
// the table action popover (e.g. the header button on the deployment
// detail page) can share the same rule.
export function isCancellableDeploymentStatus(status: DeploymentStatus): boolean {
  return CANCELLABLE_STATUSES.includes(status);
}

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
  // Redeploy: available for ready, idle, failed, or superseded deployments
  const canRedeploy =
    isActionable ||
    status === "stopped" ||
    status === "failed" ||
    status === "superseded" ||
    status === "cancelled";
  // Cancel: available for any in-flight deployment.
  const canCancel = isCancellableDeploymentStatus(status);

  return { canRollback, canPromote, canRedeploy, canCancel };
}
