"use client";

import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import { cn } from "@/lib/utils";
import { Loading } from "@unkey/ui";

const SPINNING = new Set<DeploymentStatus>([
  "starting",
  "building",
  "deploying",
  "network",
  "finalizing",
]);

const DOT_CLASS: Record<DeploymentStatus, string> = {
  pending: "bg-gray-9",
  starting: "bg-info-9",
  building: "bg-info-9",
  deploying: "bg-info-9",
  network: "bg-info-9",
  finalizing: "bg-info-9",
  ready: "bg-success-9",
  failed: "bg-error-9",
  skipped: "bg-gray-9",
  awaiting_approval: "bg-warning-9",
  stopped: "bg-gray-9",
  superseded: "bg-gray-9",
  cancelled: "bg-gray-9",
};

export function DeploymentStatusDot({ status }: { status: DeploymentStatus }) {
  return (
    <span
      className={cn(
        "size-2 shrink-0 rounded-full",
        DOT_CLASS[status],
        SPINNING.has(status) && "motion-safe:animate-pulse",
      )}
    />
  );
}

export function DeploymentStatusIndicator({ status }: { status: DeploymentStatus }) {
  if (SPINNING.has(status)) {
    return <Loading size={12} className="shrink-0 text-accent-12" />;
  }
  return <DeploymentStatusDot status={status} />;
}
