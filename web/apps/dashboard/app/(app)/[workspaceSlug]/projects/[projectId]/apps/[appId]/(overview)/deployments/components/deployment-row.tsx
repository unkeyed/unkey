"use client";

import { LastExitBadge } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/active-deployment-card";
import { DeploymentStatusDot } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/deployment-status-dot";
import { EnvironmentBadge } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/environment-badge";
import type { Deployment, Environment } from "@/lib/collections";
import { DEPLOYMENT_STATUS_LABELS } from "@/lib/collections/deploy/deployment-status";
import { shortenId } from "@/lib/shorten-id";
import { ResourceListItem } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import { useState } from "react";
import { DeploymentApproval } from "../[deploymentId]/(deployment-progress)/deployment-approval";
import { DeploymentDuration } from "./deployment-duration";
import {
  AuthorCell,
  BranchCell,
  CommitSha,
  ImageRef,
  OriginCell,
  RowMenu,
  RowTime,
} from "./deployment-row-cells";
import { imageDisplay } from "./image-reference";
import { EnvStatusBadge } from "./table/components/env-status-badge";

type DeploymentRowProps = {
  deployment: Deployment;
  environment: Environment | undefined;
  repoFullName: string | null;
  currentDeployment: Deployment | undefined;
  isRolledBack: boolean;
  href: Route;
};

export function DeploymentRow({
  deployment,
  environment,
  repoFullName,
  currentDeployment,
  isRolledBack,
  href,
}: DeploymentRowProps) {
  const [approvalOpen, setApprovalOpen] = useState(false);
  const needsApproval = deployment.status === "awaiting_approval";
  const isCurrent = currentDeployment?.id === deployment.id;
  const statusLabel = DEPLOYMENT_STATUS_LABELS[deployment.status];
  const showLastExit =
    deployment.lastExit !== null &&
    deployment.status !== "ready" &&
    deployment.status !== "superseded";

  return (
    <ResourceListItem className="flex items-center gap-3 overflow-hidden px-4 py-2.5 transition-colors hover:bg-grayA-2">
      {needsApproval ? (
        <button
          type="button"
          onClick={() => setApprovalOpen(true)}
          className="absolute inset-0 z-10 cursor-pointer"
          aria-label={`Authorize deployment ${shortenId(deployment.id)}`}
        />
      ) : (
        <Link
          href={href}
          className="absolute inset-0 z-10"
          aria-label={`Deployment ${shortenId(deployment.id)} ${statusLabel}`}
        />
      )}
      {needsApproval && (
        <DeploymentApproval
          isOpen={approvalOpen}
          onClose={() => setApprovalOpen(false)}
          deployment={deployment}
        />
      )}

      <span className="flex min-w-0 flex-1 items-center gap-2">
        <span
          className="min-w-0 truncate text-[13px] text-accent-12"
          title={deployment.gitCommitMessage ?? deployment.image ?? undefined}
        >
          {deployment.gitCommitMessage ??
            (deployment.image ? imageDisplay(deployment.image) : shortenId(deployment.id))}
        </span>
        {showLastExit && deployment.lastExit && (
          <span className="relative z-20 shrink-0">
            <LastExitBadge lastExit={deployment.lastExit} />
          </span>
        )}
      </span>

      <span className="flex min-w-0 shrink-0 items-center gap-2 md:w-44">
        <DeploymentStatusDot status={deployment.status} />
        <span className="truncate text-[13px] font-medium text-accent-12">{statusLabel}</span>
        <DeploymentDuration
          status={deployment.status}
          createdAt={deployment.createdAt}
          buildEndedAt={deployment.buildEndedAt}
        />
      </span>
      <span className="hidden w-32 shrink-0 items-center gap-2 sm:flex">
        {environment && <EnvironmentBadge environment={environment} isCurrent={isCurrent} />}
        {isCurrent && isRolledBack && <EnvStatusBadge variant="rolledBack" text="Rolled Back" />}
      </span>
      <span className="hidden w-32 min-w-0 shrink-0 items-center md:flex">
        {deployment.gitCommitSha ? (
          <CommitSha deployment={deployment} repoFullName={repoFullName} />
        ) : deployment.image ? (
          <ImageRef image={deployment.image} />
        ) : null}
      </span>
      <span className="hidden w-40 min-w-0 shrink-0 items-center lg:flex">
        {deployment.gitBranch ? (
          <BranchCell branch={deployment.gitBranch} repoFullName={repoFullName} />
        ) : (
          <OriginCell deployment={deployment} />
        )}
      </span>
      <span className="ml-auto flex shrink-0 items-center gap-3">
        <span className="hidden w-36 shrink-0 justify-end md:flex">
          <RowTime value={deployment.createdAt} />
        </span>
        <span className="hidden w-5 shrink-0 justify-center md:flex">
          <AuthorCell deployment={deployment} />
        </span>
        <RowMenu
          deployment={deployment}
          environment={environment}
          currentDeployment={currentDeployment}
          isRolledBack={isRolledBack}
        />
      </span>
    </ResourceListItem>
  );
}
