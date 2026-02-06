import type { HealthStatus } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/deployments/[deploymentId]/network/unkey-flow/components/nodes/types";
import type { Instance, Sentinel } from "@/lib/db";

export const flagCodes = ["us", "de", "au", "jp", "in", "br"] as const;
export type FlagCode = (typeof flagCodes)[number];

export function mapInstanceStatusToHealth(status: Instance["status"]): HealthStatus {
  switch (status) {
    case "running":
      return "normal";
    case "failed":
      return "unhealthy";
    case "pending":
      return "health_syncing";
    case "inactive":
      return "disabled";
  }
}

export function calculateSentinelHealth(
  instances: Partial<Instance>[],
  sentinel: Sentinel["health"],
): HealthStatus {
  // Trust the sentinel's own health check as source of truth
  if (sentinel === "unhealthy") {
    return "unhealthy";
  }

  if (sentinel === "paused") {
    return "disabled";
  }

  if (sentinel === "unknown") {
    return "unknown";
  }

  // Only consider instances for "syncing" state
  if (instances.some((i) => i.status === "pending")) {
    return "health_syncing";
  }

  return "normal";
}

export function mapRegionToFlag(region: string): FlagCode {
  const prefixMap = [
    ["ap-southeast", "au"],
    ["ap-northeast", "jp"],
    ["ap-south", "in"],
    ["us-", "us"],
    ["eu-", "de"],
    ["sa-", "br"],
  ] as const;

  return (prefixMap.find(([prefix]) => region.startsWith(prefix))?.[1] as FlagCode) ?? "us";
}
