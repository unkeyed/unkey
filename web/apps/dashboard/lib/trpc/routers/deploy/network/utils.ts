import type { HealthStatus } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/deployments/[deploymentId]/network/unkey-flow/components/nodes/types";
import type { Instance } from "@/lib/db";

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
