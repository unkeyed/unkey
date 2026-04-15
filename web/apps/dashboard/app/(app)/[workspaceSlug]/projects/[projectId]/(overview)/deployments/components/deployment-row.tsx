"use client";

import type { Deployment, Environment } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import { CodeBranch, CodeCommit } from "@unkey/icons";
import { TimestampInfo } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import dynamic from "next/dynamic";
import Link from "next/link";
import { DeploymentStatusBadge } from "../../../components/deployment-status-badge";
import { Avatar } from "../../../components/git-avatar";
import { EnvStatusBadge } from "./table/components/env-status-badge";
import { ActionColumnSkeleton } from "./table/components/skeletons";

const DeploymentListTableActions = dynamic(
  () =>
    import("./table/components/actions/deployment-list-table-action.popover.constants").then(
      (mod) => mod.DeploymentListTableActions,
    ),
  {
    loading: () => <ActionColumnSkeleton />,
    ssr: false,
  },
);

type DeploymentRowProps = {
  deployment: Deployment;
  environment?: Environment;
  isCurrent: boolean;
  isRolledBack: boolean;
  href: string;
};

export function DeploymentRow({
  deployment,
  environment,
  isCurrent,
  isRolledBack,
  href,
}: DeploymentRowProps) {
  return (
    <div
      className={cn(
        "relative flex flex-col md:flex-row md:items-center px-4 py-3 gap-3 md:gap-0 transition-colors",
        "hover:bg-grayA-2",
      )}
    >
      <Link
        href={href}
        className="absolute inset-0 z-0"
        aria-label={`Deployment ${shortenId(deployment.id)} ${deployment.status}`}
      />
      {/* Identity + Status */}
      <div className="flex items-center justify-between md:contents">
        <div className="md:w-[25%] md:shrink-0 flex flex-col gap-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-mono text-[13px] text-accent-12 truncate font-bold">
              {shortenId(deployment.id)}
            </span>
            {isCurrent ? (
              <EnvStatusBadge
                variant={isRolledBack ? "rolledBack" : "current"}
                text={isRolledBack ? "Rolled Back" : "Current"}
              />
            ) : null}
          </div>
          <span className="text-xs text-gray-9 capitalize">{environment?.slug}</span>
        </div>

        <div className="md:w-[15%] md:shrink-0">
          <DeploymentStatusBadge status={deployment.status} />
        </div>
      </div>

      {/* Source */}
      <div className="md:w-[30%] md:shrink-0 flex flex-col gap-1 min-w-0">
        <div className="flex items-center gap-2 min-w-0">
          <CodeBranch iconSize="sm-regular" className="text-accent-12 shrink-0" />
          <span
            className="font-mono text-xs text-accent-12 truncate leading-4"
            title={deployment.gitBranch}
          >
            {deployment.gitBranch}
          </span>
          {deployment.gitCommitSha ? (
            <span className="font-mono text-xs shrink-0 leading-4 -ml-1">
              <span className="text-gray-9">·</span>
              <span className="text-accent-12 ml-0.5">{deployment.gitCommitSha.slice(0, 7)}</span>
            </span>
          ) : null}
        </div>
        {deployment.gitCommitMessage ? (
          <div className="flex items-center gap-2 min-w-0">
            <CodeCommit iconSize="sm-regular" className="text-accent-12 shrink-0" />
            <span
              className="truncate text-xs text-accent-12 leading-4"
              title={deployment.gitCommitMessage}
            >
              {deployment.gitCommitMessage}
            </span>
          </div>
        ) : null}
      </div>

      {/* Meta */}
      <div className="md:w-[30%] md:shrink-0 flex items-center md:justify-end gap-3">
        <TimestampInfo
          value={deployment.createdAt}
          displayType="relative"
          className="text-[13px] text-gray-9"
        />
        <Avatar
          src={deployment.gitCommitAuthorAvatarUrl}
          alt={deployment.gitCommitAuthorHandle ?? "Author"}
        />
        <div className="z-10" role="presentation">
          <DeploymentListTableActions selectedDeployment={deployment} environment={environment} />
        </div>
      </div>
    </div>
  );
}
