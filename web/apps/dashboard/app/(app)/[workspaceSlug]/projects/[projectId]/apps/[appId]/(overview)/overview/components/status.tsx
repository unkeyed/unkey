import type { Deployment } from "@/lib/collections/deploy/deployments";
import { cn } from "@/lib/utils";

export type DeploymentDisplayStatus = "live" | "deploying" | "crashing" | "failed" | "stopped";

const DEPLOYING_STATUSES = new Set([
  "pending",
  "starting",
  "building",
  "deploying",
  "network",
  "finalizing",
  "awaiting_approval",
]);

export function deriveProductionStatus(deployment: Deployment): DeploymentDisplayStatus {
  if (deployment.status === "stopped") {
    return "stopped";
  }
  if (deployment.status === "failed") {
    return "failed";
  }
  const crashing =
    deployment.lastExit?.statusReason === "CrashLoopBackOff" ||
    (deployment.instances ?? []).some((i) => i.status === "failed");
  if (crashing) {
    return "crashing";
  }
  if (DEPLOYING_STATUSES.has(deployment.status)) {
    return "deploying";
  }
  return "live";
}

export const STATUS_META: Record<DeploymentDisplayStatus, { label: string; dotClass: string }> = {
  live: { label: "Live", dotClass: "bg-success-9" },
  deploying: { label: "Deploying", dotClass: "bg-warning-9" },
  crashing: { label: "Crashing", dotClass: "bg-error-9" },
  failed: { label: "Failed", dotClass: "bg-error-9" },
  stopped: { label: "Stopped", dotClass: "bg-gray-9" },
};

export function StatusDot({ status }: { status: DeploymentDisplayStatus }) {
  const meta = STATUS_META[status];
  return <span className={cn("size-2 shrink-0 rounded-full", meta.dotClass)} />;
}
