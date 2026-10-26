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

type DotSize = "sm" | "md";

const DOT_SIZE: Record<DotSize, { dot: string; spinner: number }> = {
  sm: { dot: "size-1.5", spinner: 10 },
  md: { dot: "size-2", spinner: 12 },
};

export function StatusDot({
  colorClass,
  pulse = false,
  size = "md",
}: {
  colorClass: string;
  pulse?: boolean;
  size?: DotSize;
}) {
  return (
    <span
      className={cn(
        "shrink-0 rounded-full",
        DOT_SIZE[size].dot,
        colorClass,
        pulse && "motion-safe:animate-pulse",
      )}
    />
  );
}

export function DeploymentStatusDot({
  status,
  size,
}: {
  status: DeploymentStatus;
  size?: DotSize;
}) {
  return (
    <StatusDot colorClass={deploymentStatusColor(status)} pulse={isBuilding(status)} size={size} />
  );
}

export function DeploymentStatusIndicator({
  status,
  size = "md",
}: {
  status: DeploymentStatus;
  size?: DotSize;
}) {
  if (isBuilding(status)) {
    return <Loading size={DOT_SIZE[size].spinner} className="shrink-0 text-gray-12" />;
  }
  return <DeploymentStatusDot status={status} size={size} />;
}

export function StatusLabel({ className, children }: PropsWithChildren<{ className?: string }>) {
  return (
    <span className={cn("flex items-center gap-2 text-sm text-gray-12", className)}>
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
