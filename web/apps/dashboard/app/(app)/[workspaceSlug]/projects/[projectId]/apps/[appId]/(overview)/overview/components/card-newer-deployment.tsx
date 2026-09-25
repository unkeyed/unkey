"use client";

import type { Deployment } from "@/lib/collections";
import {
  DEPLOYMENT_STATUS_LABELS,
  type DeploymentStatusGroup,
  statusGroupOf,
} from "@/lib/collections/deploy/deployment-status";
import { Button } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import { DeploymentStatusIndicator } from "../../../components/deployment-status-dot";
import { useBuildDuration } from "../../deployments/components/deployment-duration";
import { AuthorCell, RowTime } from "../../deployments/components/deployment-row-cells";

const TRIGGER_LABEL: Record<Deployment["trigger"], string> = {
  github: "Deployment from GitHub",
  api: "Deployment created with the API",
  cli: "Deployment created with the Unkey CLI",
  dashboard: "Deployment created from the dashboard",
  unkey: "Deployment created by Unkey",
  unknown: "New deployment",
};

const VISIBLE_BUILD_GROUPS = new Set<DeploymentStatusGroup>([
  "queued",
  "building",
  "failed",
  "blocked",
]);

export function hasVisibleBuildState(deployment: Deployment): boolean {
  return VISIBLE_BUILD_GROUPS.has(statusGroupOf(deployment.status));
}

const ACTION_LABEL: Partial<Record<DeploymentStatusGroup, string>> = {
  failed: "View build error",
  blocked: "Review build",
};

export function NewerDeploymentRow({ deployment, href }: { deployment: Deployment; href: Route }) {
  const buildTime = useBuildDuration(
    deployment.status,
    deployment.createdAt,
    deployment.buildEndedAt,
  );

  const description =
    deployment.source === "git" && deployment.gitCommitMessage
      ? deployment.gitCommitMessage
      : TRIGGER_LABEL[deployment.trigger];

  return (
    <div className="flex items-center justify-between gap-3 border-t px-4 py-2.5">
      <div className="flex min-w-0 items-center gap-2 text-[13px]">
        <DeploymentStatusIndicator status={deployment.status} />
        <span className="shrink-0 text-gray-12">{DEPLOYMENT_STATUS_LABELS[deployment.status]}</span>
        <span aria-hidden className="shrink-0 text-gray-9">
          ·
        </span>
        <span className="min-w-0 truncate text-gray-11">{description}</span>
      </div>
      <div className="flex shrink-0 items-center gap-2 text-xs text-gray-11">
        {buildTime ? (
          <span className="text-[13px] tabular-nums text-gray-9">{buildTime}</span>
        ) : (
          <RowTime value={deployment.createdAt} />
        )}
        <AuthorCell deployment={deployment} withHandle />
        <Button variant="ghost" size="sm" render={<Link href={href} />} className="shrink-0">
          {ACTION_LABEL[statusGroupOf(deployment.status)] ?? "View build"}
        </Button>
      </div>
    </div>
  );
}
