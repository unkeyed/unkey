import { DeploymentStatusIndicator } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/deployment-status-dot";
import { DEPLOYMENT_STATUS_LABELS } from "@/lib/collections/deploy/deployment-status";
import { IconCodeCommitOutline18, IconHeartPulseOutline18 } from "@unkey/icons";
import { InfoTooltip, Skeleton } from "@unkey/ui";
import Link from "next/link";
import type { ReactNode } from "react";
import type { AppRowData } from "./app-row-model";
import { AppActionsButton, DeployedAgo, LinkOrText, SourceIcon, SourceLabel } from "./app-source";

function Line({ icon, children }: { icon: ReactNode; children: ReactNode }) {
  return (
    <div className="flex min-h-5 min-w-0 items-center gap-2 text-xs text-gray-12">
      <span className="flex size-3 shrink-0 items-center justify-center text-gray-11">{icon}</span>
      {children}
    </div>
  );
}

export function AppCard({ row, projectId }: { row: AppRowData; projectId: string }) {
  const { app, deployment } = row;

  return (
    <div className="relative flex h-full w-full flex-col gap-4 rounded-lg border bg-raised p-5 shadow-xs transition-all duration-300 hover:border-strong [&_a]:z-10 [&_button]:z-10">
      <Link href={row.href} className="absolute inset-0 z-0" aria-label={`View ${app.name}`} />

      <div className="flex items-center gap-2.5">
        <span className="flex size-8 shrink-0 items-center justify-center rounded-lg border bg-raised">
          <SourceIcon source={row.source} className="size-3.5 text-gray-12" />
        </span>
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <InfoTooltip content={app.name} asChild position={{ align: "start", side: "top" }}>
            <Link
              href={row.href}
              className="min-w-0 truncate text-sm font-medium leading-4 text-gray-12"
            >
              {app.name}
            </Link>
          </InfoTooltip>
          {app.domain ? (
            <a
              href={`https://${app.domain}`}
              target="_blank"
              rel="noopener noreferrer"
              className="relative min-w-0 truncate text-xs leading-3 text-gray-11 hover:text-gray-12 hover:underline"
            >
              {app.domain}
            </a>
          ) : (
            <span className="text-xs leading-3 text-gray-9">No domain yet</span>
          )}
        </div>
        <div className="relative shrink-0">
          <AppActionsButton projectId={projectId} appId={app.id} />
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <Line icon={<SourceIcon source={row.source} className="size-3" />}>
          <SourceLabel
            row={row}
            className="relative block min-w-0 truncate text-gray-12 hover:underline"
          />
        </Line>
        {row.source === "git" ? (
          <Line icon={<IconCodeCommitOutline18 className="size-3" />}>
            {deployment?.commitMessage ? (
              <LinkOrText
                href={row.commitUrl}
                className="relative block min-w-0 truncate text-gray-12 hover:underline"
              >
                {deployment.commitMessage}
              </LinkOrText>
            ) : (
              <span className="text-gray-9">No commits deployed</span>
            )}
          </Line>
        ) : null}
        <Line
          icon={
            deployment ? (
              <DeploymentStatusIndicator status={deployment.status} />
            ) : (
              <IconHeartPulseOutline18 className="size-3" />
            )
          }
        >
          {deployment ? (
            <span className="flex min-w-0 items-center gap-1.5 whitespace-nowrap">
              <span>{DEPLOYMENT_STATUS_LABELS[deployment.status]}</span>
              <span aria-hidden="true" className="text-gray-8">
                ·
              </span>
              <DeployedAgo value={deployment.deployedAt} className="text-gray-11" />
            </span>
          ) : (
            <span className="text-gray-9">Never deployed</span>
          )}
        </Line>
      </div>
    </div>
  );
}

export function AppCardSkeleton() {
  return (
    <div className="flex h-full w-full flex-col gap-4 rounded-lg border bg-raised p-5 shadow-xs">
      <div className="flex items-center gap-2.5">
        <Skeleton className="size-8 shrink-0 rounded-lg" />
        <div className="flex flex-1 flex-col gap-1.5">
          <Skeleton className="h-3.5 w-24" />
          <Skeleton className="h-3 w-36" />
        </div>
      </div>
      <div className="flex flex-col gap-2.5">
        <Skeleton className="h-3 w-32" />
        <Skeleton className="h-3 w-44" />
        <Skeleton className="h-3 w-28" />
      </div>
    </div>
  );
}
