"use client";

import { githubUrl } from "@/lib/github-url";
import { CodeBranch, CodeCommit } from "@unkey/icons";
import { Badge, TimestampInfo } from "@unkey/ui";
import type { ReactNode } from "react";
import { MetadataCell } from "../../../components/active-deployment-card/components/metadata-cell";
import { DottedLink } from "../../../components/dotted-link";
import { Avatar } from "../../../components/git-avatar";
import { RegionFlag } from "../../../components/region-flag";
import { useProductionCard } from "./production-card-context";
import { STATUS_META, StatusDot } from "./status";

function GitHubLink({ href, children }: { href: string | undefined; children: ReactNode }) {
  if (!href) {
    return <>{children}</>;
  }
  return (
    <DottedLink href={href} external>
      {children}
    </DottedLink>
  );
}

function StatusCell() {
  const { status, isRolledBack } = useProductionCard();
  if (isRolledBack) {
    return (
      <span className="flex items-center gap-2 text-[13px] text-accent-12">
        <StatusDot status="live" />
        Live
        <Badge variant="warning" size="sm">
          Rolled back
        </Badge>
      </span>
    );
  }
  return (
    <span className="flex items-center gap-2 text-[13px] text-accent-12">
      <StatusDot status={status} />
      {STATUS_META[status].label}
    </span>
  );
}

function SourceCell() {
  const { deployment, sourceRepo, isRolledBack, rolledBackFrom } = useProductionCard();
  return (
    <div className="flex flex-col gap-1 min-w-0">
      {deployment.gitBranch && (
        <GitHubLink href={githubUrl.branch(sourceRepo, deployment.gitBranch)}>
          <span className="flex items-center gap-1.5">
            <CodeBranch iconSize="sm-regular" className="text-accent-12 shrink-0" />
            <span className="font-mono text-[13px] text-accent-12 truncate max-w-40">
              {deployment.gitBranch}
            </span>
          </span>
        </GitHubLink>
      )}
      {deployment.gitCommitSha && (
        <div className="flex items-center gap-1.5 min-w-0">
          <GitHubLink href={githubUrl.commit(sourceRepo, deployment.gitCommitSha)}>
            <span className="flex items-center gap-1.5">
              <CodeCommit iconSize="sm-regular" className="text-accent-12 shrink-0" />
              <span className="font-mono text-[13px] text-accent-12">
                {deployment.gitCommitSha.slice(0, 7)}
              </span>
            </span>
          </GitHubLink>
          {deployment.gitCommitMessage && (
            <span className="text-[13px] text-accent-12 truncate min-w-0">
              {deployment.gitCommitMessage}
            </span>
          )}
        </div>
      )}
      {isRolledBack && rolledBackFrom && (
        <div className="flex items-center gap-1.5 min-w-0 text-gray-9">
          <CodeCommit iconSize="sm-regular" className="text-gray-9 shrink-0" />
          <span className="font-mono text-[13px] line-through shrink-0">
            {rolledBackFrom.commitSha ? rolledBackFrom.commitSha.slice(0, 7) : "—"}
          </span>
          {rolledBackFrom.commitMessage && (
            <span className="text-[13px] line-through truncate min-w-0">
              {rolledBackFrom.commitMessage}
            </span>
          )}
        </div>
      )}
      {!deployment.gitBranch && !deployment.gitCommitSha && (
        <span className="font-mono text-[13px] text-accent-12 truncate">
          {deployment.image ?? "—"}
        </span>
      )}
    </div>
  );
}

export function ProductionCardMetadata() {
  const { deployment } = useProductionCard();

  const instances = deployment.instances ?? [];
  const runningCount = instances.filter((i) => i.status === "running").length;
  const regions =
    instances.length > 0
      ? [...new Map(instances.map((i) => [i.region.id, i])).values()]
      : deployment.desiredRegions;

  return (
    <div className="grid grid-cols-2 gap-y-4 gap-x-6 items-start">
      <MetadataCell label="Status">
        <StatusCell />
      </MetadataCell>

      <MetadataCell label="Region">
        {regions.length > 0 ? (
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
            {regions.map((r) => (
              <span
                key={r.region.id}
                className="flex items-center gap-1.5 text-[13px] text-accent-12"
              >
                <RegionFlag flagCode={r.flagCode} size="xs" shape="circle" />
                {r.region.name}
              </span>
            ))}
          </div>
        ) : (
          <span className="text-gray-9 text-[13px]">—</span>
        )}
      </MetadataCell>

      <MetadataCell label="Resources">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-[13px] text-gray-9">
          <span>
            <span className="text-accent-12 tabular-nums">{deployment.cpuMillicores / 1000}</span>{" "}
            vCPU
          </span>
          <span aria-hidden>·</span>
          <span>
            <span className="text-accent-12 tabular-nums">{deployment.memoryMib}</span> MiB
          </span>
        </div>
      </MetadataCell>

      <MetadataCell label="Instances">
        <span className="text-[13px] text-gray-9">
          <span className="text-accent-12 tabular-nums">{runningCount}</span> running
        </span>
      </MetadataCell>

      <MetadataCell label="Source">
        <SourceCell />
      </MetadataCell>

      <MetadataCell label="Created">
        <div className="flex items-center gap-2">
          <Avatar src={deployment.gitCommitAuthorAvatarUrl} alt="Author" />
          {deployment.gitCommitAuthorHandle && (
            <span className="font-medium text-accent-12 text-[13px] truncate">
              {deployment.gitCommitAuthorHandle}
            </span>
          )}
          <TimestampInfo
            value={deployment.createdAt}
            displayType="relative"
            className="text-gray-9 text-[13px] shrink-0"
          />
        </div>
      </MetadataCell>
    </div>
  );
}
