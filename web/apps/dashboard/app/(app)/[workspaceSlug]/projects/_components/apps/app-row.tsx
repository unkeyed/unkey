"use client";

import { DeploymentStatusLabel } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/deployment-status-dot";
import { DottedLink } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/dotted-link";
import type { ProjectApp } from "@/lib/collections/deploy/projects";
import {
  IconClockOutline18,
  IconCodeBranchOutline18,
  IconCodeCommitOutline18,
  IconEarthOutline18,
  type IconProps,
} from "@unkey/icons";
import { InfoHoverCard } from "@unkey/ui";
import { cn } from "cn";
import Link from "next/link";
import type { ComponentPropsWithRef, FC, ReactElement, ReactNode } from "react";
import { type AppDeployment, DeploymentMeta, useDeploymentPhrase } from "./deployment-meta";

// base-ui merges the handlers, aria wiring and ref it needs into this element,
// so swallowing props silently detaches the hover.
export function AppRow({
  app,
  className,
  ...props
}: ComponentPropsWithRef<typeof Link> & { app: ProjectApp }) {
  return (
    <Link
      {...props}
      className={cn(
        "flex min-w-0 items-center gap-2 rounded-md transition-colors hover:bg-grayA-3",
        className,
      )}
    >
      <span className="min-w-0 flex-1 truncate text-sm font-medium leading-5 text-gray-12">
        {app.name}
      </span>
      {app.headlineDeployment ? <DeploymentMeta deployment={app.headlineDeployment} /> : null}
    </Link>
  );
}

export function AppDetailHoverCard({
  app,
  width,
  children,
}: { app: ProjectApp; width?: number; children: ReactElement }) {
  if (!app.headlineDeployment) {
    return children;
  }

  return (
    <InfoHoverCard
      asChild
      delayDuration={100}
      style={{ width }}
      position={{ side: "right", align: "center" }}
      content={<AppDetail app={app} deployment={app.headlineDeployment} />}
    >
      {children}
    </InfoHoverCard>
  );
}

function AppDetail({ app, deployment }: { app: ProjectApp; deployment: AppDeployment }) {
  const deployedPhrase = useDeploymentPhrase(deployment);

  const rows: [string, FC<IconProps>, ReactNode][] = [];
  if (deployment.commitMessage) {
    rows.push(["commit", IconCodeCommitOutline18, deployment.commitMessage]);
  }
  if (deployment.branch) {
    rows.push(["branch", IconCodeBranchOutline18, deployment.branch]);
  }
  if (app.customDomain) {
    rows.push([
      "domain",
      IconEarthOutline18,
      <DottedLink
        key="domain"
        href={`https://${app.customDomain}`}
        external
        className="min-w-0 truncate font-mono text-xs font-medium text-gray-12 underline decoration-dotted underline-offset-3 transition-all hover:decoration-solid"
      >
        {app.customDomain}
      </DottedLink>,
    ]);
  }
  rows.push(["deployed", IconClockOutline18, deployedPhrase]);

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <span className="min-w-0 truncate text-sm font-medium text-gray-12">{app.name}</span>
        <DeploymentStatusLabel
          status={deployment.status}
          className="ml-auto shrink-0 text-xs text-gray-11"
        />
      </div>
      <div className="flex flex-col gap-1 font-mono text-xs text-gray-11">
        {rows.map(([key, Icon, value]) => (
          <span key={key} className="flex items-center gap-2">
            <Icon className="size-3 shrink-0 text-gray-9" />
            <span className="min-w-0 truncate">{value}</span>
          </span>
        ))}
      </div>
    </div>
  );
}
