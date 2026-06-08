import type { HealthStatus } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/deployments/[deploymentId]/network/unkey-flow/components/nodes/types";
import type { Instance } from "@/lib/db";

export const flagCodes = ["us", "de", "au", "jp", "in", "br", "local"] as const;
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

export function mapRegionToFlag(region: string): FlagCode {
  const prefixMap = [
    ["ap-southeast", "au"],
    ["ap-northeast", "jp"],
    ["ap-south", "in"],
    ["us-", "us"],
    ["eu-", "de"],
    ["sa-", "br"],
    ["local", "local"],
  ] as const;

  return (prefixMap.find(([prefix]) => region.startsWith(prefix))?.[1] as FlagCode) ?? "us";
}
