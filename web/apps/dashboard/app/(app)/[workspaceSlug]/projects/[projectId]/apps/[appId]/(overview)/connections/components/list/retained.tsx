"use client";

import { EnvironmentLabel } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import { IconCubeOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import { ResourceListBody, ResourceListContent, ResourceListItem } from "@unkey/ui";
import type { RetainedView } from "../../hooks/use-connections-data";
import type { ConnectionEnvs } from "../../model/merge";

export function RetainedConnections({
  retained,
  envs,
}: {
  retained: RetainedView;
  envs: ConnectionEnvs;
}) {
  return match(retained)
    .with({ type: "loading" }, () => null)
    .with({ type: "error" }, () => (
      <p role="alert" className="text-sm text-error-11">
        We couldn't load connections saved in existing deployments. Reload to try again.
      </p>
    ))
    .with({ type: "ready" }, ({ rows }) =>
      rows.length === 0 ? null : (
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
                  key={`${row.env}-${row.id}`}
                  className="grid h-12 grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,1.2fr)_32px] items-center gap-4 px-4"
                >
                  <span className="flex min-w-0 items-center gap-2">
                    <IconCubeOutline18 className="size-3.5 shrink-0 text-gray-9" />
                    <span className="truncate text-sm font-medium text-gray-11">
                      {row.targetAppName}
                    </span>
                  </span>
                  <code className="truncate font-mono text-xs text-gray-9">{row.id}</code>
                  <span className="flex items-center gap-2 text-xs text-gray-11">
                    <EnvironmentLabel
                      environment={{ slug: envs[row.env]?.slug ?? row.env, kind: row.env }}
                    />
                    · {row.deployments} saved {row.deployments === 1 ? "deployment" : "deployments"}
                  </span>
                  <span />
                </ResourceListItem>
              ))}
            </ResourceListBody>
          </ResourceListContent>
        </section>
      ),
    )
    .exhaustive();
}
