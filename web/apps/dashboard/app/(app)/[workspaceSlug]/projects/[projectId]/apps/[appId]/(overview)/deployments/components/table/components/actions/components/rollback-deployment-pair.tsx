"use client";

import { Avatar } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/git-avatar";
import type { Deployment } from "@/lib/collections";
import { deploymentTitle } from "@/lib/collections/deploy/deployment-title";
import {
  IconArrowDotRotateAnticlockwiseOutline12,
  IconArrowDottedRotateAnticlockwiseOutline18,
  IconCircleXmarkOutline18,
  IconCloudOutline12,
  IconCodeBranchOutline18,
  IconCodeCommitOutline18,
} from "@unkey/icons";
import { TimestampInfo } from "@unkey/ui";
import type { ReactNode } from "react";

type RollbackPairProps = {
  current: Deployment;
  target: Deployment;
};

function Meta({ deployment }: { deployment: Deployment }) {
  return (
    <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-12">
      {deployment.gitCommitSha && (
        <span className="flex items-center gap-1.5">
          <IconCodeCommitOutline18 className="size-3 shrink-0" />
          <span className="font-mono">{deployment.gitCommitSha.slice(0, 7)}</span>
        </span>
      )}
      {deployment.gitBranch && (
        <span className="flex min-w-0 items-center gap-1.5">
          <IconCodeBranchOutline18 className="size-3 shrink-0" />
          <span className="truncate font-mono">{deployment.gitBranch}</span>
        </span>
      )}
      {deployment.gitCommitAuthorHandle && (
        <span className="flex min-w-0 items-center gap-1.5">
          <Avatar
            src={deployment.gitCommitAuthorAvatarUrl}
            alt={deployment.gitCommitAuthorHandle}
            className="size-4"
          />
          <span className="truncate">{deployment.gitCommitAuthorHandle}</span>
        </span>
      )}
      <span>
        Deployed{" "}
        <TimestampInfo value={deployment.createdAt} displayType="relative" className="inline" />
      </span>
    </div>
  );
}

function RollbackBadge({ kind }: { kind: "current" | "previous" | "newer" }) {
  return kind === "current" ? (
    <span className="inline-flex h-5.5 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-md border px-2 text-xs leading-none border-transparent bg-info-11 text-white dark:bg-info-9 dark:text-gray-1">
      <IconCloudOutline12 className="shrink-0" />
      Current
    </span>
  ) : (
    <span className="inline-flex h-5.5 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-md border px-2 text-xs leading-none text-gray-12">
      <IconArrowDotRotateAnticlockwiseOutline12 className="shrink-0" />
      {kind === "previous" ? "Previous" : "Newer"}
    </span>
  );
}

function Row({
  deployment,
  icon,
  badge,
}: {
  deployment: Deployment;
  icon: ReactNode;
  badge: ReactNode;
}) {
  return (
    <div className="flex items-start gap-3">
      <span className="mt-0.5 shrink-0">{icon}</span>
      <div className="flex min-w-0 flex-1 flex-col gap-1.5">
        <div className="flex items-start justify-between gap-3">
          <span className="min-w-0 truncate text-[13px] font-medium text-gray-12">
            {deploymentTitle(deployment)}
          </span>
          <span className="shrink-0">{badge}</span>
        </div>
        <Meta deployment={deployment} />
      </div>
    </div>
  );
}

export function RollbackDeploymentPair({ current, target }: RollbackPairProps) {
  return (
    <>
      <div className="rounded-lg border bg-raised px-3.5 py-3">
        <Row
          deployment={current}
          icon={<IconCircleXmarkOutline18 className="size-4 text-gray-11" />}
          badge={<RollbackBadge kind="current" />}
        />
      </div>
      <h3 className="mt-3 text-[13px] text-gray-11">To this deployment</h3>
      <div className="rounded-lg border bg-raised px-3.5 py-3">
        <Row
          deployment={target}
          icon={<IconArrowDottedRotateAnticlockwiseOutline18 className="size-4 text-gray-12" />}
          badge={
            <RollbackBadge kind={target.createdAt < current.createdAt ? "previous" : "newer"} />
          }
        />
      </div>
    </>
  );
}
