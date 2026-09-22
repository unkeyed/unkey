"use client";

import {
  DEPLOYMENT_STATUS_LABELS,
  type DeploymentStatus,
  deploymentStatusColor,
  statusGroupOf,
} from "@/lib/collections/deploy/deployment-status";
import { Loading } from "@unkey/ui";
import { cn } from "cn";
import type { PropsWithChildren } from "react";

function isBuilding(status: DeploymentStatus): boolean {
  return statusGroupOf(status) === "building";
}

export function StatusDot({ colorClass, pulse = false }: { colorClass: string; pulse?: boolean }) {
  return (
    <span
      className={cn(
        "size-2 shrink-0 rounded-full",
        colorClass,
        pulse && "motion-safe:animate-pulse",
      )}
    />
  );
}

export function DeploymentStatusDot({ status }: { status: DeploymentStatus }) {
  return <StatusDot colorClass={deploymentStatusColor(status)} pulse={isBuilding(status)} />;
}

export function DeploymentStatusIndicator({ status }: { status: DeploymentStatus }) {
  if (isBuilding(status)) {
    return <Loading size={12} className="shrink-0 text-gray-12" />;
  }
  return <DeploymentStatusDot status={status} />;
}

export function StatusLabel({ className, children }: PropsWithChildren<{ className?: string }>) {
  return (
    <span className={cn("flex items-center gap-2 text-[13px] text-gray-12", className)}>
      {children}
    </span>
  );
}

export function DeploymentStatusLabel({
  status,
  className,
}: { status: DeploymentStatus; className?: string }) {
  return (
    <StatusLabel className={className}>
      <DeploymentStatusIndicator status={status} />
      {DEPLOYMENT_STATUS_LABELS[status]}
    </StatusLabel>
  );
}
