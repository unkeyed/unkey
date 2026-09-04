"use client";

import { githubUrl } from "@/lib/github-url";
import { CodeBranch, CodeCommit, Layers2 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Badge, CopyButton, InfoTooltip, TimestampInfo } from "@unkey/ui";
import type { ReactNode } from "react";
import { MetadataCell } from "../../../components/active-deployment-card/components/metadata-cell";
import { DeploymentStatusBadge } from "../../../components/deployment-status-badge";
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
  const { deployment, status, isCurrent, isRolledBack } = useProductionCard();
  if (!isCurrent) {
    return <DeploymentStatusBadge status={deployment.status} />;
  }
  if (isRolledBack) {
    return (
      <span className="flex items-center gap-2 text-[13px] text-accent-12">
        <StatusDot status={status} />
        {STATUS_META[status].label}
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
      {match(deployment.source)
        .with("git", () => (
          <>
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
          </>
        ))
        .with("oci", () => (
          <span className="flex items-center gap-1.5 min-w-0">
            <Layers2 iconSize="sm-regular" className="text-accent-12 shrink-0" />
            <span
              className="font-mono text-[13px] text-accent-12 truncate"
              title={deployment.requestedImage ?? deployment.resolvedImage ?? undefined}
            >
              {deployment.requestedImage ?? deployment.resolvedImage ?? "No image available"}
            </span>
            {deployment.resolvedImage && (
              <InfoTooltip content="Copy resolved image" asChild>
                <CopyButton
                  value={deployment.resolvedImage}
                  variant="ghost"
                  className="size-5 shrink-0"
                  toastMessage={deployment.resolvedImage}
                  src="production-deployment-source"
                />
              </InfoTooltip>
            )}
          </span>
        ))
        .with("unknown", () => (
          <span className="flex items-center gap-1.5 min-w-0 text-gray-9">
            <Layers2 iconSize="sm-regular" className="shrink-0" />
            <span className="text-[13px]">Unknown source</span>
          </span>
        ))
        .exhaustive()}
      {isRolledBack && rolledBackFrom && (
        <div className="flex items-center gap-1.5 min-w-0 text-gray-9 line-through">
          {match(rolledBackFrom.source)
            .with("git", () => (
              <>
                <CodeCommit iconSize="sm-regular" className="shrink-0" />
                <span className="font-mono text-[13px] shrink-0">
                  {rolledBackFrom.gitCommitSha?.slice(0, 7) ?? "—"}
                </span>
                {rolledBackFrom.gitCommitMessage && (
                  <span className="text-[13px] truncate min-w-0">
                    {rolledBackFrom.gitCommitMessage}
                  </span>
                )}
              </>
            ))
            .with("oci", () => (
              <>
                <Layers2 iconSize="sm-regular" className="shrink-0" />
                <span className="font-mono text-[13px] truncate min-w-0">
                  {rolledBackFrom.requestedImage ?? rolledBackFrom.resolvedImage ?? "Unknown image"}
                </span>
              </>
            ))
            .with("unknown", () => (
              <>
                <Layers2 iconSize="sm-regular" className="shrink-0" />
                <span className="text-[13px]">Unknown source</span>
              </>
            ))
            .exhaustive()}
        </div>
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
        {match(deployment.source)
          .with("git", () => (
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
          ))
          .with("oci", "unknown", () => (
            <TimestampInfo
              value={deployment.createdAt}
              displayType="relative"
              className="text-gray-9 text-[13px] shrink-0"
            />
          ))
          .exhaustive()}
      </MetadataCell>
    </div>
  );
}
