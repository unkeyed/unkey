import { DeploymentStatusIndicator } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/deployment-status-dot";
import { DEPLOYMENT_STATUS_LABELS } from "@/lib/collections/deploy/deployment-status";
import {
  InfoTooltip,
  ResourceListBody,
  ResourceListContent,
  ResourceListItem,
  Skeleton,
} from "@unkey/ui";
import Link from "next/link";
import type { AppRowData } from "./app-row-model";
import { AppActionsButton, DeployedAgo, LinkOrText, SourceIcon, SourceLabel } from "./app-source";

export function AppsTable({ rows, projectId }: { rows: AppRowData[]; projectId: string }) {
  return (
    <ResourceListContent>
      <div className="overflow-x-auto">
        <div className="min-w-[760px]">
          <AppsTableHeader />
          <ResourceListBody>
            {rows.map((row) => (
              <AppsTableRow key={row.app.id} row={row} projectId={projectId} />
            ))}
          </ResourceListBody>
        </div>
      </div>
    </ResourceListContent>
  );
}

export function AppsTableSkeleton() {
  return (
    <ResourceListContent aria-busy="true">
      <div className="overflow-x-auto">
        <div className="min-w-[760px]">
          <AppsTableHeader />
          <ResourceListBody aria-hidden="true">
            {["skeleton-1", "skeleton-2", "skeleton-3"].map((key) => (
              <ResourceListItem key={key} className="flex h-12 items-center gap-4 px-4">
                <Skeleton className="size-6 shrink-0 rounded-md" />
                <Skeleton className="h-3 w-28" />
                <Skeleton className="h-3 w-32" />
                <Skeleton className="h-3 w-40" />
              </ResourceListItem>
            ))}
          </ResourceListBody>
        </div>
      </div>
    </ResourceListContent>
  );
}

function AppsTableHeader() {
  return (
    <div className="grid grid-cols-[minmax(0,1.2fr)_minmax(0,1.2fr)_minmax(0,1.5fr)_minmax(0,0.8fr)_minmax(0,0.9fr)_32px] items-center gap-4 border-b bg-table-header px-4 py-[7px] text-xs font-medium text-gray-12">
      <span>App</span>
      <span>Repository</span>
      <span>Domain</span>
      <span>Branch</span>
      <span>Deployed</span>
      <span />
    </div>
  );
}

function AppsTableRow({ row, projectId }: { row: AppRowData; projectId: string }) {
  const { app, deployment } = row;

  return (
    <ResourceListItem className="grid grid-cols-[minmax(0,1.2fr)_minmax(0,1.2fr)_minmax(0,1.5fr)_minmax(0,0.8fr)_minmax(0,0.9fr)_32px] items-center gap-4 h-12 px-4 text-xs text-gray-12 transition-colors hover:bg-grayA-2">
      <Link href={row.href} className="absolute inset-0 z-0" aria-label={`View ${app.name}`} />
      <span className="flex min-w-0 items-center gap-2.5">
        <span className="flex size-6 shrink-0 items-center justify-center rounded-md border bg-raised">
          <SourceIcon source={row.source} className="size-3 text-gray-12" />
        </span>
        <span className="truncate text-[13px] font-medium">{app.name}</span>
      </span>
      <span className="flex min-w-0 items-center gap-1.5">
        <SourceIcon source={row.source} className="size-3 shrink-0 text-gray-11" />
        <SourceLabel row={row} className="relative z-10 min-w-0 truncate hover:underline" />
      </span>
      <span className="min-w-0">
        {app.domain ? (
          <LinkOrText
            href={`https://${app.domain}`}
            className="relative z-10 block truncate font-medium hover:underline"
          >
            {app.domain}
          </LinkOrText>
        ) : (
          <span className="text-gray-9">—</span>
        )}
      </span>
      <span className="min-w-0">
        {row.source === "git" && deployment?.branch ? (
          <LinkOrText
            href={row.branchUrl}
            className="relative z-10 inline-block max-w-full truncate rounded-sm bg-gray-3 px-1 align-middle font-mono text-[11px] leading-4 ring-1 ring-grayA-4"
          >
            {deployment.branch}
          </LinkOrText>
        ) : (
          <span className="text-gray-9">—</span>
        )}
      </span>
      <span className="flex min-w-0 items-center gap-1.5">
        {deployment ? (
          <>
            <InfoTooltip
              content={DEPLOYMENT_STATUS_LABELS[deployment.status]}
              asChild
              position={{ side: "top" }}
            >
              <span className="relative z-10 flex">
                <DeploymentStatusIndicator status={deployment.status} />
              </span>
            </InfoTooltip>
            <span className="sr-only">{DEPLOYMENT_STATUS_LABELS[deployment.status]}</span>
            <DeployedAgo value={deployment.deployedAt} className="truncate" />
          </>
        ) : (
          <span className="text-gray-9">Never</span>
        )}
      </span>
      <span className="relative z-10 flex justify-end">
        <AppActionsButton projectId={projectId} appId={app.id} />
      </span>
    </ResourceListItem>
  );
}
