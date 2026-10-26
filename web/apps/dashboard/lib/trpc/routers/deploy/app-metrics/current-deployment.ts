import {
  type DeploymentStatus,
  isDeploymentInFlight,
} from "@/lib/collections/deploy/deployment-status";
import type { DeployRole } from "@/lib/metrics/app-metrics";
import type { ProductionRouting } from "./scope";

export type LiveDeployment = { id: string; status: DeploymentStatus; createdAt: number };

export function currentDeploymentId(input: {
  routing: Pick<ProductionRouting, "currentDeploymentId"> | null;
  live: ReadonlyArray<LiveDeployment>;
}): string | null {
  if (input.routing) {
    return input.routing.currentDeploymentId;
  }
  return (
    input.live
      .filter((d) => d.status === "ready")
      .toSorted((a, b) => b.createdAt - a.createdAt || b.id.localeCompare(a.id))
      .at(0)?.id ?? null
  );
}

export function deployRole(
  deployment: Pick<LiveDeployment, "id" | "status">,
  currentId: string | null,
): DeployRole {
  if (deployment.id === currentId) {
    return "current";
  }
  return isDeploymentInFlight(deployment.status) ? "rolling-out" : "previous";
}
