"use client";

import { EnvStatusBadge } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/deployments/components/table/components/env-status-badge";
import { RegionFlag } from "@/components/region-flag";
import { collection } from "@/lib/collections";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import { summarizeCrashLoop } from "@/lib/collections/deploy/exit-reasons";
import { githubUrl } from "@/lib/github-url";
import {
  formatCpu,
  formatCpuParts,
  formatMemory,
  formatMemoryParts,
  formatStorage,
  formatStorageParts,
} from "@/lib/utils/deployment-formatters";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { IconCodeBranchOutline18, IconCodeCommitOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Card, InfoTooltip, TimestampInfo } from "@unkey/ui";
import { useProjectData } from "../../(overview)/data-provider";
import { DeploymentTriggerBadge } from "../../../../components/deployment-trigger-badge";
import { Avatar } from "../../components/git-avatar";
import { DottedLink } from "../dotted-link";
import { LastExitText } from "../last-exit";
import { ActiveDeploymentCardEmpty } from "./components/active-deployment-card-empty";
import { ImageSource } from "./components/image-source";
import { MetadataCell } from "./components/metadata-cell";

function GitHubLink({ href, children }: { href: string | undefined; children: React.ReactNode }) {
  if (!href) {
    return children;
  }
  return (
    <DottedLink href={href} external>
      {children}
    </DottedLink>
  );
}

type ActiveDeploymentCardProps = {
  deployment: Deployment;
  statusBadge?: React.ReactNode;
  expandableContent?: React.ReactNode;
  isCurrent?: boolean;
  isRolledBack?: boolean;
  environmentSlug?: string;
};

export function ActiveDeploymentCard({
  deployment,
  statusBadge,
  expandableContent,
  isCurrent,
  isRolledBack,
  environmentSlug,
}: ActiveDeploymentCardProps) {
  const { projectId } = useProjectData();
  // Repo connections are per-app; the project-level repositoryFullName is
  // just some app's connection in this project.
  const appsQuery = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId)),
    [projectId],
  );
  const repoFullName = (appsQuery.data ?? []).find(
    (a) => a.id === deployment?.appId,
  )?.repositoryFullName;
  const sourceRepo = deployment?.forkRepositoryFullName || repoFullName;

  if (!deployment) {
    return <ActiveDeploymentCardEmpty />;
  }

  const cpu = formatCpuParts(deployment.cpuMillicores);
  const mem = formatMemoryParts(deployment.memoryMib);
  const storage = deployment.storageMib > 0 ? formatStorageParts(deployment.storageMib) : null;
  const actualInstances = deployment.instances ?? [];
  const hasActualInstances = actualInstances.length > 0;
  const runningCount = actualInstances.filter((i) => i.status === "running").length;
  const targetCount = deployment.desiredInstanceCount;

  const uniqueRegions = hasActualInstances
    ? [...new Map(actualInstances.map((i) => [i.region.id, i])).values()]
    : deployment.desiredRegions;
  return (
    <Card className="flex flex-col">
      <div className="px-4 pt-3 pb-2.5">
        <div className="flex w-full justify-between items-center gap-4">
          <div className="flex items-baseline gap-2">
            <span className="font-mono text-sm text-gray-12 font-semibold shrink-0">
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
            {match(deployment.source)
              .with("git", () =>
                deployment.gitCommitMessage ? (
                  <GitHubLink href={githubUrl.commit(sourceRepo, deployment.gitCommitSha)}>
                    <div className="flex items-center gap-1.5 min-w-0">
                      <IconCodeCommitOutline18 className="size-3 text-gray-12 shrink-0" />
                      <span className="text-xs text-gray-12 truncate">
                        {deployment.gitCommitMessage}
                      </span>
                    </div>
                  </GitHubLink>
                ) : null,
              )
              .with("oci", "unknown", () => null)
              .exhaustive()}
            {deployment.crashLoop && (
              <LastExitText summary={summarizeCrashLoop(deployment.crashLoop)} />
            )}
            {statusBadge}
          </div>
        </div>
      </div>

      <div className="border-t px-4 py-3">
        <div className="grid grid-cols-2 md:grid-cols-3 gap-y-4 gap-x-6 items-start">
          <MetadataCell label="Created">
            {match(deployment.source)
              .with("git", () => (
                <div className="flex items-center gap-2">
                  <Avatar src={deployment.gitCommitAuthorAvatarUrl} alt="Author" />
                  {deployment.gitCommitAuthorHandle && (
                    <>
                      <span data-sentry-mask className="font-medium text-gray-12 text-xs">
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
              ))
              .with("oci", "unknown", () => (
                <TimestampInfo
                  value={deployment.createdAt}
                  displayType="relative"
                  className="text-gray-9 text-xs self-start text-left"
                />
              ))
              .exhaustive()}
          </MetadataCell>

          <MetadataCell label="Source">
            <div className="flex items-center gap-2 min-w-0">
              {match(deployment.source)
                .with("oci", () => (
                  <ImageSource
                    image={deployment.requestedImage ?? deployment.resolvedImage}
                    copyValue={deployment.resolvedImage}
                  />
                ))
                .with("git", () => (
                  <>
                    {deployment.gitBranch && (
                      <GitHubLink href={githubUrl.branch(sourceRepo, deployment.gitBranch)}>
                        <span className="flex items-center gap-1">
                          <IconCodeBranchOutline18 className="size-3 text-gray-12 shrink-0" />
                          <span className="font-mono text-xs text-gray-12 truncate max-w-32">
                            {deployment.gitBranch}
                          </span>
                        </span>
                      </GitHubLink>
                    )}
                    {deployment.gitCommitSha && (
                      <>
                        {deployment.gitBranch && <span className="text-gray-9 text-xs">·</span>}
                        <GitHubLink href={githubUrl.commit(sourceRepo, deployment.gitCommitSha)}>
                          <span className="flex items-center gap-1">
                            {!deployment.gitBranch && (
                              <IconCodeCommitOutline18 className="size-3 text-gray-12 shrink-0" />
                            )}
                            <span className="font-mono text-xs text-gray-12">
                              {deployment.gitCommitSha.slice(0, 7)}
                            </span>
                          </span>
                        </GitHubLink>
                      </>
                    )}
                  </>
                ))
                .with("unknown", () => <ImageSource image={null} />)
                .exhaustive()}
            </div>
          </MetadataCell>

          {environmentSlug && (
            <MetadataCell label="Environment">
              <span className="text-xs text-gray-12 capitalize">{environmentSlug}</span>
            </MetadataCell>
          )}

          {deployment.trigger !== "unknown" && (
            <MetadataCell label="Triggered By">
              <DeploymentTriggerBadge
                trigger={deployment.trigger}
                triggeredBy={deployment.triggeredBy}
                triggerReason={deployment.triggerReason}
              />
            </MetadataCell>
          )}

          <MetadataCell label="Resources">
            <div className="flex items-center gap-2 text-xs">
              <InfoTooltip
                content={`CPU: ${formatCpu(deployment.cpuMillicores)}`}
                position={{ side: "top", align: "center" }}
              >
                <span>
                  <span className="font-medium text-gray-12">{cpu.value}</span>{" "}
                  <span className="text-gray-11">{cpu.unit}</span>
                </span>
              </InfoTooltip>
              <span className="text-gray-9">·</span>
              <InfoTooltip
                content={`Memory: ${formatMemory(deployment.memoryMib)}`}
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
                    content={`Storage: ${formatStorage(deployment.storageMib)}`}
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
            <span className="font-medium text-gray-12 text-xs">
              {`${runningCount} of ${targetCount}`}
            </span>
          </MetadataCell>

          <MetadataCell label="Regions">
            <div className="flex items-center gap-2 text-xs">
              {uniqueRegions.length > 0 ? (
                <div className="flex items-center gap-1.5">
                  {uniqueRegions.map((instance) => (
                    <InfoTooltip
                      key={instance.region.id}
                      content={instance.region.name}
                      position={{ side: "top", align: "center" }}
                    >
                      <RegionFlag region={instance.region.name} size="xs" shape="rounded" />
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
