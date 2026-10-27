import { useWarmAppHandlers } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/hooks/use-warm-app-page";
import { DeploymentStatusIndicator } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/deployment-status-dot";
import { SourceIcon } from "@/components/deploy/app-source";
import { ListTableRow, listTable } from "@/components/list-views";
import { DEPLOYMENT_STATUS_LABELS } from "@/lib/collections/deploy/deployment-status";
import { InfoTooltip, badgeVariants } from "@unkey/ui";
import { cn } from "cn";
import type { AppRowData } from "./app-row-model";
import { AppActionsButton, DeployedAgo, LinkOrText, SourceLabel } from "./app-source";

export const APPS_TABLE = listTable({
  minWidth: 760,
  columns: [
    { id: "app", header: "App", width: "minmax(0,1.2fr)", skeleton: "title" },
    { id: "repository", header: "Repository", width: "minmax(0,1.2fr)", skeleton: "text" },
    { id: "domain", header: "Domain", width: "minmax(0,1.5fr)", skeleton: "text" },
    { id: "branch", header: "Branch", width: "minmax(0,0.8fr)", skeleton: "text" },
    { id: "deployed", header: "Deployed", width: "minmax(0,0.9fr)", skeleton: "text" },
    { id: "actions", header: null, width: "32px", skeleton: "none" },
  ],
});

export function AppsTableRow({
  row,
  projectId,
  table,
}: {
  row: AppRowData;
  projectId: string;
  table: typeof APPS_TABLE;
}) {
  const { app, deployment } = row;
  const warm = useWarmAppHandlers(projectId, app.id);

  return (
    <ListTableRow
      table={table}
      href={row.href}
      label={`View ${app.name}`}
      {...warm}
      cells={{
        app: (
          <span className="flex min-w-0 items-center gap-2.5">
            <span className="flex size-6 shrink-0 items-center justify-center rounded-md border bg-raised">
              <SourceIcon source={row.source} className="size-3 text-gray-12" />
            </span>
            <span className="truncate text-sm font-medium">{app.name}</span>
          </span>
        ),
        repository: (
          <span className="flex min-w-0 items-center gap-1.5">
            <SourceIcon source={row.source} className="size-3 shrink-0 text-gray-11" />
            <SourceLabel row={row} className="relative z-10 min-w-0 truncate hover:underline" />
          </span>
        ),
        domain: (
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
        ),
        branch: (
          <span className="min-w-0">
            {row.source === "git" && deployment?.branch ? (
              <LinkOrText
                href={row.branchUrl}
                className={cn(badgeVariants({ variant: "code" }), "relative z-10 text-inherit")}
              >
                {deployment.branch}
              </LinkOrText>
            ) : (
              <span className="text-gray-9">—</span>
            )}
          </span>
        ),
        deployed: (
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
        ),
        actions: (
          <span className="relative z-10 flex justify-end">
            <AppActionsButton projectId={projectId} appId={app.id} />
          </span>
        ),
      }}
    />
  );
}
