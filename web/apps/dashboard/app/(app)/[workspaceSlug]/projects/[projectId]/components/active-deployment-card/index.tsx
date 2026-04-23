"use client";

import { EnvStatusBadge } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/deployments/components/table/components/env-status-badge";
import {
  formatCpuParts,
  formatMemoryParts,
  formatStorageParts,
} from "@/lib/utils/deployment-formatters";
import { CodeBranch, CodeCommit } from "@unkey/icons";
import { InfoTooltip, TimestampInfo } from "@unkey/ui";
import { Card } from "../../(overview)/components/card";
import { useProjectData } from "../../(overview)/data-provider";
import { Avatar } from "../../components/git-avatar";
import { RegionFlag } from "../../components/region-flag";
import { ActiveDeploymentCardEmpty } from "./components/active-deployment-card-empty";
import { MetadataCell } from "./components/metadata-cell";
import { ActiveDeploymentCardSkeleton } from "./components/skeleton";
import { DottedLink } from "../dotted-link";

function GitHubLink({ href, children }: { href: string | undefined; children: React.ReactNode }) {
  if (!href) {
    return children;
  }
  return (
    <DottedLink
      href={href}
      external
    >
      {children}
    </DottedLink>
  );
}

type ActiveDeploymentCardProps = {
  deploymentId: string | null;
  statusBadge?: React.ReactNode;
  expandableContent?: React.ReactNode;
  isCurrent?: boolean;
  isRolledBack?: boolean;
  environmentSlug?: string;
};

export function ActiveDeploymentCard({
  deploymentId,
  statusBadge,
  expandableContent,
  isCurrent,
  isRolledBack,
  environmentSlug,
}: ActiveDeploymentCardProps) {
  const { getDeploymentById, isDeploymentsLoading, project } = useProjectData();
  const deployment = deploymentId ? getDeploymentById(deploymentId) : undefined;
  const repoFullName = project?.repositoryFullName;
  const sourceRepo = deployment?.forkRepositoryFullName || repoFullName;

  if (isDeploymentsLoading) {
    return <ActiveDeploymentCardSkeleton />;
  }
  if (!deployment) {
    return <ActiveDeploymentCardEmpty />;
  }

  const cpu = formatCpuParts(deployment.cpuMillicores);
  const mem = formatMemoryParts(deployment.memoryMib);
  const storage = deployment.storageMib > 0 ? formatStorageParts(deployment.storageMib) : null;
  const instances = deployment.instances ?? [];
  const uniqueRegions = [...new Map(instances.map((i) => [i.region.id, i])).values()];

  return (
    <Card className="flex flex-col">
      <div className="px-4 pt-3 pb-2.5">
        <div className="flex w-full justify-between items-center gap-4">
          <div className="flex items-baseline gap-2">
            <span className="font-mono text-[13px] text-accent-12 font-semibold shrink-0">
              {deployment.id}
            </span>
            {isCurrent && (
              <EnvStatusBadge
                variant={isRolledBack ? "rolledBack" : "current"}
                text={isRolledBack ? "Rolled Back" : "Current"}
              />
            )}
          </div>
          <div className="flex items-center gap-3 min-w-0">
            {deployment.gitCommitMessage && (
              <GitHubLink
                href={
                  deployment.gitCommitSha && sourceRepo
                    ? `https://github.com/${sourceRepo}/commit/${deployment.gitCommitSha}`
                    : undefined
                }
              >
                <div className="flex items-center gap-1.5 min-w-0">
                  <CodeCommit iconSize="sm-regular" className="text-accent-12 shrink-0" />
                  <span className="text-xs text-accent-12 truncate">
                    {deployment.gitCommitMessage}
                  </span>
                </div>
              </GitHubLink>
            )}
            {statusBadge}
          </div>
        </div>
      </div>

      <div className="border-t border-gray-4 px-4 py-3">
        <div className="grid grid-cols-2 md:grid-cols-3 gap-y-4 gap-x-6 items-start">
          <MetadataCell label="Created">
            <div className="flex items-center gap-2">
              <Avatar src={deployment.gitCommitAuthorAvatarUrl} alt="Author" />
              {deployment.gitCommitAuthorHandle && (
                <>
                  <span className="font-medium text-accent-12 text-xs">
                    {deployment.gitCommitAuthorHandle}
                  </span>
                  <span className="text-gray-9 text-xs">·</span>
                </>
              )}
              <TimestampInfo
                value={deployment.createdAt}
                displayType="relative"
                className="text-gray-9 text-xs"
              />
            </div>
          </MetadataCell>

          <MetadataCell label="Source">
            <div className="flex items-center gap-2 min-w-0">
              {deployment.gitBranch && (
                <GitHubLink
                  href={
                    sourceRepo
                      ? `https://github.com/${sourceRepo}/tree/${deployment.gitBranch}`
                      : undefined
                  }
                >
                  <span className="flex items-center gap-1">
                    <CodeBranch iconSize="sm-regular" className="text-accent-12 shrink-0" />
                    <span className="font-mono text-xs text-accent-12 truncate max-w-32">
                      {deployment.gitBranch}
                    </span>
                  </span>
                </GitHubLink>
              )}
              {deployment.gitCommitSha && (
                <>
                  <span className="text-gray-9 text-xs">·</span>
                  <GitHubLink
                    href={
                      sourceRepo
                        ? `https://github.com/${sourceRepo}/commit/${deployment.gitCommitSha}`
                        : undefined
                    }
                  >
                    <span className="font-mono text-xs text-accent-12">
                      {deployment.gitCommitSha.slice(0, 7)}
                    </span>
                  </GitHubLink>
                </>
              )}
            </div>
          </MetadataCell>

          {environmentSlug && (
            <MetadataCell label="Environment">
              <span className="text-xs text-accent-12 capitalize">{environmentSlug}</span>
            </MetadataCell>
          )}

          <MetadataCell label="Resources">
            <div className="flex items-center gap-2 text-xs">
              <InfoTooltip
                content={`CPU: ${cpu.value} ${cpu.unit}`}
                variant="inverted"
                position={{ side: "top", align: "center" }}
              >
                <span>
                  <span className="font-medium text-gray-12">{cpu.value}</span>{" "}
                  <span className="text-gray-11">{cpu.unit}</span>
                </span>
              </InfoTooltip>
              <span className="text-gray-9">·</span>
              <InfoTooltip
                content={`Memory: ${mem.value} ${mem.unit}`}
                variant="inverted"
                position={{ side: "top", align: "center" }}
              >
                <span>
                  <span className="font-medium text-gray-12">{mem.value}</span>{" "}
                  <span className="text-gray-11">{mem.unit}</span>
                </span>
              </InfoTooltip>
              {storage && (
                <>
                  <span className="text-gray-9">·</span>
                  <InfoTooltip
                    content={`Storage: ${storage.value} ${storage.unit}`}
                    variant="inverted"
                    position={{ side: "top", align: "center" }}
                  >
                    <span>
                      <span className="font-medium text-gray-12">{storage.value}</span>{" "}
                      <span className="text-gray-11">{storage.unit} Disk</span>
                    </span>
                  </InfoTooltip>
                </>
              )}
            </div>
          </MetadataCell>

          <MetadataCell label="Instances">
            <span className="font-medium text-gray-12 text-xs">{instances.length}</span>
          </MetadataCell>

          <MetadataCell label="Regions">
            <div className="flex items-center gap-2 text-xs">
              {uniqueRegions.length > 0 ? (
                <div className="flex items-center gap-1.5">
                  {uniqueRegions.map((instance) => (
                    <InfoTooltip
                      key={instance.region.id}
                      content={instance.region.name}
                      variant="inverted"
                      position={{ side: "top", align: "center" }}
                    >
                      <RegionFlag flagCode={instance.flagCode} size="xs" shape="rounded" />
                    </InfoTooltip>
                  ))}
                </div>
              ) : (
                <span className="text-gray-11">—</span>
              )}
            </div>
          </MetadataCell>
        </div>
      </div>
      {expandableContent}
    </Card>
  );
}
