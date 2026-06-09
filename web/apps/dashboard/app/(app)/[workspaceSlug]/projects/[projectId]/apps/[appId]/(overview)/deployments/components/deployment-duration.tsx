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
  updatedAt: number | null;
};

export function DeploymentDuration({ status, createdAt, updatedAt }: Props) {
  const display = STATUS_DISPLAY[status];
  const isLive = display === "live";

  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (!isLive) {
      return;
    }
    const interval = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(interval);
  }, [isLive]);

  if (display === "hidden") {
    return null;
  }

  const endTime = isLive ? now : updatedAt;
  if (endTime === null) {
    return null;
  }

  return (
    <span className="flex items-center gap-1 text-xs font-mono text-gray-9">
      {isLive && <Loading size={14} className="text-accent-12" />}
      {formatCompoundDuration(Math.max(0, endTime - createdAt))}
    </span>
  );
}
