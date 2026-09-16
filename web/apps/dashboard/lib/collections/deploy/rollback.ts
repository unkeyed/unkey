import type { DeploymentStatus } from "./deployment-status";

type RollbackSibling = {
  id: string;
  environmentId: string;
  status: DeploymentStatus;
  createdAt: number;
};

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
