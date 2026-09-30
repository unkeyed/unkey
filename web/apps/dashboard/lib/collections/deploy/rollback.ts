import type { DeploymentStatus } from "./deployment-status";

type RollbackSibling = {
  id: string;
  environmentId: string;
  status: DeploymentStatus;
  createdAt: number;
};

type TrafficState = {
  status: DeploymentStatus;
  desiredState: "running" | "stopped";
};

type RollbackCandidate = RollbackSibling & TrafficState;

export function isRollbackTarget(deployment: TrafficState): boolean {
  return deployment.status === "ready" && deployment.desiredState === "running";
}

export function rollbackCandidates<T extends RollbackCandidate>(
  deployments: readonly T[],
  current: RollbackSibling,
): T[] {
  return deployments
    .filter(
      (d) =>
        d.environmentId === current.environmentId && d.id !== current.id && isRollbackTarget(d),
    )
    .sort((a, b) => b.createdAt - a.createdAt);
}

export function previousRollbackTarget<T extends RollbackCandidate>(
  deployments: readonly T[],
  current: RollbackSibling,
): T | undefined {
  return rollbackCandidates(deployments, current).find((d) => d.createdAt < current.createdAt);
}

// Nothing records which deployment a rollback replaced, so it is inferred as the
// newest ready deployment in the same environment that came after the live one.
export function findRolledBackFrom<T extends RollbackSibling>(
  deployments: readonly T[],
  current: RollbackSibling,
): T | undefined {
  return deployments
    .filter(
      (d) =>
        d.environmentId === current.environmentId &&
        d.status === "ready" &&
        d.createdAt > current.createdAt,
    )
    .sort((a, b) => b.createdAt - a.createdAt)[0];
}
