"use client";

import type { Deployment } from "@/lib/collections/deploy/deployments";
import { findRolledBackFrom } from "@/lib/collections/deploy/rollback";
import { githubUrl } from "@/lib/github-url";
import { IconArrowUpRightOutline12 } from "@unkey/icons";
import {
  Card,
  TimestampInfo,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
  buttonVariants,
} from "@unkey/ui";
import type { ReactNode } from "react";
import { StatusDot } from "../../../../components/deployment-status-dot";
import { DottedLink } from "../../../../components/dotted-link";
import { EnvironmentBadge, badgeRollout } from "../../../../components/environment-badge";
import { useProjectData } from "../../../data-provider";
import { useAppCurrentDeployment } from "../../../hooks/use-app-current-deployment";
import { useRecentDeployments } from "../../../hooks/use-recent-deployments";
import { useBuildDuration } from "../../components/deployment-duration";
import { InstantRollbackRow } from "../instant-rollback-row";
import type { DeploymentPage } from "../use-deployment-page";
import { runningCountText } from "./deployment-health";
import { DomainSummary } from "./domain-summary";

type DeploymentDetailsCardProps = {
  page: Pick<
    DeploymentPage,
    "deployment" | "status" | "pageStatus" | "instances" | "domains" | "domainsLoading"
  >;
};

export function DeploymentDetailsCard({ page }: DeploymentDetailsCardProps) {
  const { deployment, status, pageStatus, instances, domainsLoading } = page;
  const { primary, additional } = page.domains;
  const { environments } = useProjectData();
  const { app, currentDeployment, isRolledBack } = useAppCurrentDeployment();
  const { deployments } = useRecentDeployments({ enabled: isRolledBack });
  const isRolledBackFrom =
    isRolledBack && currentDeployment
      ? findRolledBackFrom(deployments, currentDeployment)?.id === deployment.id
      : false;
  const repoFullName = deployment.forkRepositoryFullName || app?.repositoryFullName || undefined;

  const environment = environments.find((e) => e.id === deployment.environmentId);
  const isCurrent = app?.currentDeploymentId === deployment.id;
  const duration = useBuildDuration(status, deployment.createdAt, deployment.buildEndedAt);

  return (
    <Card className="flex flex-col overflow-hidden">
      <div className="flex items-center gap-3 border-b border-grayA-4 px-4 py-3">
        <DomainSummary primary={primary} additional={additional} isLoading={domainsLoading} />
        <span className="ml-auto flex min-h-7 shrink-0 items-center gap-3">
          <span className="font-mono text-xs text-gray-9">{deployment.id}</span>
          {primary && (
            <a
              href={primary.url}
              target="_blank"
              rel="noopener noreferrer"
              className={buttonVariants({ variant: "outline", size: "sm" })}
            >
              Visit
              <IconArrowUpRightOutline12 className="size-3! text-gray-10" />
            </a>
          )}
        </span>
      </div>
      <div className="grid divide-y divide-grayA-4 md:grid-cols-2 md:divide-x md:divide-y-0">
        <Column title="Runtime">
          <Row label="Status">
            <StatusDot colorClass={pageStatus.dotClass} pulse={pageStatus.inFlight} />
            <span className="truncate font-medium">{pageStatus.label}</span>
            {duration !== null && (
              <span className="font-mono text-xs tabular-nums text-gray-9">{duration}</span>
            )}
          </Row>
          <Row label="Environment">
            {environment ? (
              <EnvironmentBadge
                environment={environment}
                rollout={badgeRollout({ isCurrent, isRolledBack, isRolledBackFrom })}
                liveSince={isCurrent ? (app?.updatedAt ?? null) : null}
              />
            ) : (
              <span className="text-gray-10">—</span>
            )}
          </Row>
          <Row label="Instances">{instances ? runningCountText(instances) : "—"}</Row>
        </Column>
        <Column title="Source">
          <SourceRows deployment={deployment} repoFullName={repoFullName} />
          <Row label="Created">
            {deployment.gitCommitAuthorHandle && (
              <span data-sentry-mask className="truncate">
                {deployment.gitCommitAuthorHandle}
              </span>
            )}
            <TimestampInfo
              value={deployment.createdAt}
              displayType="relative"
              className="shrink-0 text-xs text-gray-10"
            />
          </Row>
        </Column>
      </div>
      <InstantRollbackRow key={deployment.id} />
    </Card>
  );
}

function Column({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="flex min-w-0 flex-col gap-2 px-4 py-3">
      <span className="pb-1 text-xs font-medium text-gray-10">{title}</span>
      {children}
    </div>
  );
}

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex min-h-6 min-w-0 items-center gap-3">
      <span className="w-24 shrink-0 text-xs font-medium text-gray-11">{label}</span>
      <div className="flex min-w-0 items-center gap-1.5 text-sm text-gray-12">{children}</div>
    </div>
  );
}

function SourceRows({
  deployment,
  repoFullName,
}: { deployment: Deployment; repoFullName: string | undefined }) {
  if (deployment.source !== "git") {
    return (
      <Row label="Image">
        <span className="truncate font-mono text-xs">
          {deployment.requestedImage ?? deployment.resolvedImage ?? "—"}
        </span>
      </Row>
    );
  }
  const branchUrl = githubUrl.branch(repoFullName, deployment.gitBranch);
  return (
    <>
      <Row label="Branch">
        <Tooltip>
          <TooltipTrigger
            render={<span className="flex min-w-0" tabIndex={branchUrl ? undefined : 0} />}
          >
            <DottedLink href={branchUrl} external>
              <span className="block max-w-40 truncate font-mono text-xs">
                {deployment.gitBranch}
              </span>
            </DottedLink>
          </TooltipTrigger>
          <TooltipContent className="font-mono">{deployment.gitBranch}</TooltipContent>
        </Tooltip>
        {deployment.gitCommitSha && (
          <span className="flex shrink-0">
            <DottedLink href={githubUrl.commit(repoFullName, deployment.gitCommitSha)} external>
              <span className="font-mono text-xs text-gray-10">
                {deployment.gitCommitSha.slice(0, 7)}
              </span>
            </DottedLink>
          </span>
        )}
      </Row>
      <Row label="Commit">
        <span className="truncate">{deployment.gitCommitMessage ?? "—"}</span>
      </Row>
    </>
  );
}
