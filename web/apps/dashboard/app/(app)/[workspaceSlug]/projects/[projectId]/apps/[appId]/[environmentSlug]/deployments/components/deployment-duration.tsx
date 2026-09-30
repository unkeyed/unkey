"use client";

import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import { formatCompoundDuration } from "@/lib/utils/metric-formatters";
import { Loading } from "@unkey/ui";
import { useEffect, useState } from "react";

type DurationDisplay = "hidden" | "terminal" | "live";

const STATUS_DISPLAY: Record<DeploymentStatus, DurationDisplay> = {
  pending: "hidden",
  awaiting_approval: "hidden",
  starting: "live",
  building: "live",
  deploying: "live",
  network: "live",
  finalizing: "live",
  ready: "terminal",
  failed: "terminal",
  skipped: "terminal",
  stopped: "terminal",
  superseded: "terminal",
  cancelled: "terminal",
};

type Props = {
  status: DeploymentStatus;
  createdAt: number;
  // When every build step closed (from deployment_steps). Null while a build
  // is still running or when a deployment has no steps. Unlike updatedAt, stop
  // and wake never move it, so terminal rows show the build duration rather
  // than the deployment's age.
  buildEndedAt: number | null;
};

function isBuildTicking(status: DeploymentStatus, buildEndedAt: number | null): boolean {
  return STATUS_DISPLAY[status] === "live" && buildEndedAt === null;
}

export function useBuildDuration(
  status: DeploymentStatus,
  createdAt: number,
  buildEndedAt: number | null,
): string | null {
  const isTicking = isBuildTicking(status, buildEndedAt);
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (!isTicking) {
      return;
    }
    const interval = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(interval);
  }, [isTicking]);

  if (STATUS_DISPLAY[status] === "hidden") {
    return null;
  }

  // A finished build shows its immutable duration. Checked before the live
  // branch so a woken deployment (status deploying, steps long closed) shows
  // the original build time instead of ticking from its months-old createdAt.
  if (buildEndedAt !== null) {
    return formatCompoundDuration(Math.max(0, buildEndedAt - createdAt));
  }

  // A genuinely in-progress build (an open step) ticks from createdAt.
  if (isTicking) {
    return formatCompoundDuration(Math.max(0, now - createdAt));
  }

  // Terminal status with no clean end (no steps, or an abandoned build): hide
  // rather than invent a number.
  return null;
}

export function DeploymentDuration({ status, createdAt, buildEndedAt }: Props) {
  const duration = useBuildDuration(status, createdAt, buildEndedAt);

  if (duration === null) {
    return null;
  }

  return (
    <span className="flex items-center gap-1 text-xs font-mono text-gray-9">
      {isBuildTicking(status, buildEndedAt) && <Loading size={14} className="text-gray-12" />}
      {duration}
    </span>
  );
}
