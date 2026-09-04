"use client";

import { EnvironmentBadge } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/environment-badge";
import type { Deployment, Environment } from "@/lib/collections";
import { ResourceListItem } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import {
  AuthorCell,
  BranchCell,
  IdChip,
  RowMenu,
  RowTime,
  SourceChip,
} from "../../../deployments/components/deployment-row-cells";

type ActiveBranchRowProps = {
  branch: string;
  deployment: Deployment;
  environment: Environment | undefined;
  repoFullName: string | null;
  currentDeployment: Deployment | undefined;
  isRolledBack: boolean;
  href: Route;
};

export function ActiveBranchRow({
  branch,
  deployment,
  environment,
  repoFullName,
  currentDeployment,
  isRolledBack,
  href,
}: ActiveBranchRowProps) {
  return (
    <ResourceListItem className="flex items-center gap-3 overflow-hidden px-4 py-2.5 transition-colors hover:bg-grayA-2">
      <Link
        href={href}
        className="absolute inset-0 z-10"
        aria-label={`Latest deployment on ${branch}`}
      />
      <span className="flex min-w-0 flex-1 items-center">
        <BranchCell branch={branch} />
      </span>
      <span className="hidden w-24 shrink-0 items-center sm:flex">
        {environment && <EnvironmentBadge environment={environment} isCurrent={false} />}
      </span>
      <span className="flex w-32 shrink-0 items-center">
        <IdChip deployment={deployment} href={href} />
      </span>
      <span className="hidden w-28 min-w-0 shrink-0 items-center md:flex">
        <SourceChip deployment={deployment} repoFullName={repoFullName} />
      </span>
      <span className="hidden w-40 shrink-0 items-center lg:flex">
        <AuthorCell deployment={deployment} withHandle />
      </span>
      <span className="hidden w-28 shrink-0 items-center justify-end md:flex">
        <RowTime value={deployment.createdAt} />
      </span>
      <RowMenu
        deployment={deployment}
        environment={environment}
        currentDeployment={currentDeployment}
        isRolledBack={isRolledBack}
      />
    </ResourceListItem>
  );
}
