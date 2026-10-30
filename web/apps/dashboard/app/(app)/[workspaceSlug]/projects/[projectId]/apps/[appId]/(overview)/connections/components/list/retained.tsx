"use client";

import { ListPager } from "@/components/list-pager";
import { LoadError } from "@/components/load-error";
import { plural } from "@/lib/fmt";
import type { ListView } from "@/lib/loadable/list";
import { IconCubeOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import { ResourceListBody, ResourceListContent, ResourceListItem } from "@unkey/ui";
import type { RetainedConnection } from "../../hooks/use-connections-data";

export function RetainedConnections({ retained }: { retained: ListView<RetainedConnection> }) {
  return match(retained)
    .with({ status: "loading" }, { status: "empty" }, () => null)
    .with({ status: "failed" }, ({ error, retry }) => (
      <section className="mt-8">
        <LoadError
          title="Could not load connections saved in existing deployments"
          error={error}
          frame="none"
          onRetry={retry}
        />
      </section>
    ))
    .with({ status: "ready" }, ({ rows, footer }) => (
      <section className="mt-8 flex flex-col gap-3">
        <div className="flex flex-col gap-0.5">
          <h2 className="text-sm font-medium text-gray-12">Existing deployments only</h2>
          <p className="text-xs text-gray-11">
            These connections were removed. Deployments saved before that still include them.
          </p>
        </div>
        <ResourceListContent>
          <ResourceListBody>
            {rows.map((row) => (
              <ResourceListItem
                key={row.id}
                className="grid h-12 grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,1.2fr)_32px] items-center gap-4 px-4"
              >
                <span className="flex min-w-0 items-center gap-2">
                  <IconCubeOutline18 className="size-3.5 shrink-0 text-gray-9" />
                  <span className="truncate text-sm font-medium text-gray-11">
                    {row.targetAppName}
                  </span>
                </span>
                <code className="truncate font-mono text-xs text-gray-9">{row.id}</code>
                <span className="text-xs text-gray-11">
                  {plural(row.deployments, "saved deployment")}
                </span>
                <span />
              </ResourceListItem>
            ))}
          </ResourceListBody>
          <ListPager state={footer} noun="saved connections" />
        </ResourceListContent>
      </section>
    ))
    .exhaustive();
}
