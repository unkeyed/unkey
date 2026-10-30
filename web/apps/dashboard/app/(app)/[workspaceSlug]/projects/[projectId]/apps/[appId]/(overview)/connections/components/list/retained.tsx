"use client";

import { ListPager } from "@/components/list-pager";
import { LoadError } from "@/components/load-error";
import { plural } from "@/lib/fmt";
import type { ListView } from "@/lib/loadable/list";
import { IconCubeOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Badge, InfoTooltip, ResourceListItem } from "@unkey/ui";
import type { RetainedConnection } from "../../hooks/use-connections-data";

export function RemovedConnectionRows({ retained }: { retained: ListView<RetainedConnection> }) {
  return match(retained)
    .with({ status: "loading" }, { status: "empty" }, () => null)
    .with({ status: "failed" }, ({ error, retry }) => (
      <LoadError
        title="Could not load removed connections"
        error={error}
        frame="none"
        onRetry={retry}
      />
    ))
    .with({ status: "ready" }, ({ rows, footer }) => (
      <>
        {rows.map((row) => (
          <RemovedConnectionRow key={row.id} row={row} />
        ))}
        <ListPager state={footer} noun="removed connections" />
      </>
    ))
    .exhaustive();
}

function RemovedConnectionRow({ row }: { row: RetainedConnection }) {
  return (
    <ResourceListItem className="grid h-12 grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,1.2fr)_32px] items-center gap-4 px-4">
      <span className="flex min-w-0 items-center gap-2">
        <IconCubeOutline18 className="size-3.5 shrink-0 text-gray-9" />
        <span className="truncate text-sm font-medium text-gray-11">{row.targetAppName}</span>
        <Badge variant="secondary" size="sm" className="shrink-0 text-2xs">
          Removed
        </Badge>
      </span>
      <span className="text-xs text-gray-9">—</span>
      <span className="flex min-w-0">
        <InfoTooltip
          asChild
          position={{ side: "top" }}
          content={
            <span className="flex max-w-64">
              This connection was removed. Deployments saved before that still use it until they
              redeploy.
            </span>
          }
        >
          <span className="truncate text-xs text-gray-11">
            In {plural(row.deployments, "saved deployment")}
          </span>
        </InfoTooltip>
      </span>
      <span />
    </ResourceListItem>
  );
}
