"use client";

import { EnvStatusBadge } from "@/app/(app)/[workspaceSlug]/projects/[projectSlug]/apps/[appSlug]/(overview)/deployments/components/table/components/env-status-badge";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import type { LastExit } from "@/lib/types/deploy";
import {
  formatCpuParts,
  formatMemoryParts,
  formatStorageParts,
} from "@/lib/utils/deployment-formatters";
import { CodeBranch, CodeCommit } from "@unkey/icons";
import { match } from "@unkey/match";
import { Badge, InfoTooltip, TimestampInfo } from "@unkey/ui";
import { Card } from "../../(overview)/components/card";
import { useProjectData } from "../../(overview)/data-provider";
import { Avatar } from "../../components/git-avatar";
import { RegionFlag } from "../../components/region-flag";
import { DottedLink } from "../dotted-link";
import { ActiveDeploymentCardEmpty } from "./components/active-deployment-card-empty";
import { MetadataCell } from "./components/metadata-cell";
import { ActiveDeploymentCardSkeleton } from "./components/skeleton";

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
  deploymentId: string | null;
  deployment?: Deployment;
  statusBadge?: React.ReactNode;
  expandableContent?: React.ReactNode;
  isCurrent?: boolean;
  isRolledBack?: boolean;
  environmentSlug?: string;
};

export function ActiveDeploymentCard({
  deploymentId,
  deployment: directDeployment,
  statusBadge,
  expandableContent,
  isCurrent,
  isRolledBack,
  environmentSlug,
}: ActiveDeploymentCardProps) {
  const { getDeploymentById, isDeploymentsLoading, project } = useProjectData();
  const deployment =
    directDeployment ?? (deploymentId ? getDeploymentById(deploymentId) : undefined);
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
  const actualInstances = deployment.instances ?? [];
  const hasActualInstances = actualInstances.length > 0;
  const runningCount = actualInstances.filter((i) => i.status === "running").length;
  const targetCount = deployment.desiredInstanceCount;

  const uniqueRegions = hasActualInstances
    ? [...new Map(actualInstances.map((i) => [i.region.id, i])).values()]
    : deployment.desiredRegions;
  // Hide the badge once the deployment has converged on ready: an old
  // OOMKill that was resolved by the next push isn't useful context here.
  const showLastExit =
    deployment.lastExit && deployment.status !== "ready" && deployment.status !== "superseded";

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
            {showLastExit && deployment.lastExit && (
              <LastExitBadge lastExit={deployment.lastExit} />
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
                  {deployment.gitBranch && <span className="text-gray-9 text-xs">·</span>}
                  <GitHubLink
                    href={
                      sourceRepo
                        ? `https://github.com/${sourceRepo}/commit/${deployment.gitCommitSha}`
                        : undefined
                    }
                  >
                    <span className="flex items-center gap-1">
                      {!deployment.gitBranch && (
                        <CodeCommit iconSize="sm-regular" className="text-accent-12 shrink-0" />
                      )}
                      <span className="font-mono text-xs text-accent-12">
                        {deployment.gitCommitSha.slice(0, 7)}
                      </span>
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

// LastExitBadge renders a compact "OOMKilled · exit=137" pill.
// CrashLoopBackOff is warning, terminations are error.
// Exported so the deployments list row + network instance card can reuse
// it next to the status badge — same surface, same data, different page.
export function LastExitBadge({ lastExit }: { lastExit: LastExit }) {
  const isCrashloop = lastExit.statusReason === "CrashLoopBackOff";
  const reason = isCrashloop ? "CrashLoopBackOff" : (lastExit.reason ?? "Error");
  const variant = isCrashloop ? "warning" : "error";

  const tooltip = explainExit(reason, lastExit.exitCode, lastExit.signal);

  return (
    <InfoTooltip content={tooltip} variant="inverted" position={{ side: "top", align: "end" }}>
      <Badge variant={variant} className="text-xs whitespace-nowrap">
        {reason}
        {(() => {
          const showExitCode =
            !isCrashloop && lastExit.exitCode !== null && lastExit.exitCode !== 0;
          return (
            showExitCode && (
              <span className="ml-1 font-mono tabular-nums">· exit={lastExit.exitCode}</span>
            )
          );
        })()}
      </Badge>
    </InfoTooltip>
  );
}

// explainExit produces tooltip for LastExitBadge with user-friendly descriptions.
function explainExit(
  reason: string,
  exitCode: number | null,
  signal: number | null,
): React.ReactNode {
  // CrashLoopBackOff: point users at the exit code for diagnosis.
  if (reason === "CrashLoopBackOff") {
    return (
      <div className="flex flex-col gap-1.5 max-w-[280px]">
        <div className="font-medium">App keeps crashing on startup</div>
        <div>
          Your app has exited too many times in a row, so we're slowing down restart attempts to
          give it room to recover.
        </div>
        <div>
          Check the recent crash entries below for the exit code that's causing the loop, and your
          logs for the underlying error.
        </div>
      </div>
    );
  }

  const lines: { label: string; body: string }[] = [];

  const reasonLine = match(reason)
    .with("OOMKilled", () => ({
      label: "Out of memory",
      body: "Your app used more memory than its configured limit. Either it has a memory leak, or the limit is set too low for what it actually needs at peak.",
    }))
    .with("Error", () => ({
      label: "App exited with an error",
      body: "Your app stopped with a non-zero status. The exit code below narrows down how.",
    }))
    .with("ContainerCannotRun", () => ({
      label: "Couldn't start your app",
      body: "We were unable to start your app at all. Usually means the start command, entrypoint, or image is broken. Check your build settings and Dockerfile.",
    }))
    .with("Completed", () => ({
      label: "Exited cleanly",
      body: "Your app shut down without an error. Normal for one-off jobs; unusual for a service that's supposed to keep running. Check whether your main loop returned early.",
    }))
    .when(
      (r) => Boolean(r),
      (r) => ({
        label: r,
        body: "Your app exited. The exit code below has more detail.",
      }),
    )
    .otherwise(() => null);

  if (reasonLine) {
    lines.push(reasonLine);
  }

  if (exitCode !== null && exitCode !== 0) {
    const exitLine = describeExitCode(exitCode, signal);
    if (exitLine) {
      lines.push(exitLine);
    }
  }

  if (lines.length === 0) {
    return reason || "App exited";
  }

  return (
    <div className="flex flex-col gap-1.5 max-w-[280px]">
      {lines.map((line) => (
        <div key={line.label} className="flex flex-col gap-0.5">
          <div className="font-medium">{line.label}</div>
          <div>{line.body}</div>
        </div>
      ))}
    </div>
  );
}

// describeExitCode maps the exit codes users actually see to plain-
// language hints. Codes ≥128 conventionally mean "force-killed", but
// users don't need the signal arithmetic — just whether it's a memory
// issue or an app crash and what to look at next.
const EXIT_CODE_DESCRIPTIONS: Record<number, { label: string; body: string }> = {
  1: {
    label: "exit=1",
    body: "Your app returned an error. Check your logs for the cause. Typically an unhandled error or a failed startup check.",
  },
  2: {
    label: "exit=2",
    body: "Your app crashed with an unhandled error. Usually a panic or uncaught exception. Your logs should have a stack trace.",
  },
  126: {
    label: "exit=126",
    body: "We found your start command but couldn't run it. Most often a permissions issue or wrong shebang. Make sure the binary is executable.",
  },
  127: {
    label: "exit=127",
    body: "We couldn't find the command you asked us to run. Check that your start command and binary path are correct in your build settings.",
  },
  128: {
    label: "exit=128",
    body: "Your app was force-killed. Often a memory issue (same root cause as exit 137, see that one for more detail).",
  },
  137: {
    label: "exit=137",
    body: "Your app was force-killed, almost always because it ran out of memory. Either it has a leak, or you need to raise its memory limit.",
  },
  139: {
    label: "exit=139",
    body: "Your app crashed due to an invalid memory access. This is a bug in your code, usually a null pointer or buffer overrun. Check your stack trace.",
  },
  143: {
    label: "exit=143",
    body: "Your app shut down cleanly when asked to. Normal during a redeploy; unusual otherwise.",
  },
};

function describeExitCode(
  exitCode: number,
  signal: number | null,
): { label: string; body: string } | null {
  const description = EXIT_CODE_DESCRIPTIONS[exitCode];
  if (description) {
    return description;
  }

  if (signal && signal > 0) {
    return {
      label: `exit=${exitCode}`,
      body: "Your app was force-killed. Often a memory issue. Check your app's memory usage near the time of the crash.",
    };
  }

  return {
    label: `exit=${exitCode}`,
    body: "Your app exited with this code. Check your app's docs for what it means. Most apps document their exit codes.",
  };
}
