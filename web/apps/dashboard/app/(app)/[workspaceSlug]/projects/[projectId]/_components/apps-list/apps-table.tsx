import { DeploymentStatusIndicator } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/deployment-status-dot";
import { DEPLOYMENT_STATUS_LABELS } from "@/lib/collections/deploy/deployment-status";
import { InfoTooltip, ResourceListBody, ResourceListContent, ResourceListItem } from "@unkey/ui";
import { cn } from "cn";
import Link from "next/link";
import type { AppRowData } from "./app-row-model";
import { AppActionsButton, DeployedAgo, LinkOrText, SourceIcon, SourceLabel } from "./app-source";

const COLUMNS =
  "grid grid-cols-[minmax(0,1.2fr)_minmax(0,1.2fr)_minmax(0,1.5fr)_minmax(0,0.8fr)_minmax(0,0.9fr)_32px] items-center gap-4";

const empty = <span className="text-gray-9">—</span>;

export function AppsTable({ rows, projectId }: { rows: AppRowData[]; projectId: string }) {
  return (
    <ResourceListContent>
      <div
        className={cn(
          COLUMNS,
          "border-b bg-table-header px-4 py-[7px] text-xs font-medium text-gray-12",
        )}
      >
        <span>App</span>
        <span>Repository</span>
        <span>Domain</span>
        <span>Branch</span>
        <span>Deployed</span>
        <span />
      </div>
      <ResourceListBody>
        {rows.map((row) => (
          <AppsTableRow key={row.app.id} row={row} projectId={projectId} />
        ))}
      </ResourceListBody>
    </ResourceListContent>
  );
}

function AppsTableRow({ row, projectId }: { row: AppRowData; projectId: string }) {
  const { app, deployment } = row;

  return (
    <ResourceListItem
      className={cn(COLUMNS, "h-12 px-4 text-xs text-gray-12 transition-colors hover:bg-grayA-2")}
    >
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
          <a
            href={`https://${app.domain}`}
            target="_blank"
            rel="noopener noreferrer"
            className="relative z-10 block truncate font-medium hover:underline"
          >
            {app.domain}
          </a>
        ) : (
          empty
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
          empty
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
