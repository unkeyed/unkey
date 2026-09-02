"use client";

import {
  Empty,
  ResourceListBody,
  ResourceListContent,
  ResourceListItem,
  Skeleton,
} from "@unkey/ui";
import Link from "next/link";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { CreateLogdrainButton } from "./create-logdrain-button";
import { DrainRowChart } from "./drain-row-chart";
import { DrainActions, SinkType, StatusBadge } from "./logdrain-ui";

const SKELETON_ROWS = 8;

function LogdrainsSkeleton() {
  return (
    <ResourceListContent aria-busy="true" aria-live="polite">
      <output className="sr-only">Loading log drains…</output>
      <ResourceListBody aria-hidden="true">
        {Array.from({ length: SKELETON_ROWS }).map((_, index) => (
          <ResourceListItem
            // biome-ignore lint/suspicious/noArrayIndexKey: skeleton rows are static and never reorder
            key={index}
            className="flex flex-col gap-3 px-4 py-3 md:flex-row md:items-center md:gap-0"
          >
            <div className="flex min-w-0 flex-col gap-1 md:w-[30%] md:shrink-0">
              <Skeleton className="h-3.5 w-40" />
              <Skeleton className="h-3 w-24" />
            </div>
            <div className="flex items-center gap-2 md:w-[20%] md:shrink-0">
              <Skeleton className="size-5" />
              <Skeleton className="h-3 w-12" />
            </div>
            <div className="md:w-[20%] md:shrink-0">
              <Skeleton className="h-5 w-20 rounded-md" />
            </div>
            <div className="flex items-center gap-3 md:w-[30%] md:shrink-0 md:justify-end">
              <Skeleton className="h-7 w-[158px] rounded-md" />
              <Skeleton className="size-5" />
            </div>
          </ResourceListItem>
        ))}
      </ResourceListBody>
    </ResourceListContent>
  );
}

export function LogdrainsList() {
  const workspace = useWorkspaceNavigation();
  const query = trpc.logdrain.list.useQuery();
  if (query.isLoading) {
    return <LogdrainsSkeleton />;
  }
  if (query.isError) {
    return (
      <ResourceListContent aria-live="polite">
        <div className="flex w-full items-center justify-center px-4 py-16">
          <Empty>
            <Empty.Title>Unable to load log drains</Empty.Title>
            <Empty.Description>{query.error.message}</Empty.Description>
          </Empty>
        </div>
      </ResourceListContent>
    );
  }
  if (!query.data?.length) {
    return (
      <ResourceListContent aria-live="polite">
        <div className="p-3">
          <Empty className="min-h-[200px] rounded-lg border border-dashed border-gray-4 bg-gray-1/50">
            <Empty.Title>No log drains yet</Empty.Title>
            <Empty.Description className="max-w-sm">
              Create a log drain to send audit logs to an HTTPS endpoint or Axiom dataset.
            </Empty.Description>
            <Empty.Actions>
              <CreateLogdrainButton />
            </Empty.Actions>
          </Empty>
        </div>
      </ResourceListContent>
    );
  }
  return (
    <ResourceListContent aria-live="polite">
      <ResourceListBody aria-label="Log drains">
        {query.data.map((drain) => {
          return (
            <ResourceListItem
              key={drain.id}
              className="group flex flex-col gap-3 px-4 py-3 transition-colors hover:bg-grayA-2 md:flex-row md:items-center md:gap-0"
            >
              <Link
                href={routes.settings.logdrains.detail({
                  workspaceSlug: workspace.slug,
                  drainId: drain.id,
                })}
                className="absolute inset-0 z-10 rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7"
                aria-label={`Log drain ${drain.name}`}
              />
              <div className="flex min-w-0 flex-col gap-1 md:w-[30%] md:shrink-0">
                <span className="truncate text-[13px] font-semibold text-accent-12">
                  {drain.name}
                </span>
                <span className="text-xs text-gray-9">Audit logs</span>
              </div>
              <div className="flex items-center gap-2 text-[13px] font-medium text-accent-12 md:w-[20%] md:shrink-0">
                <SinkType kind={drain.kind} />
              </div>
              <div className="md:w-[20%] md:shrink-0">
                <StatusBadge status={drain.status} />
              </div>
              <div className="flex items-center gap-3 md:w-[30%] md:shrink-0 md:justify-end">
                <div className="relative z-20">
                  <DrainRowChart drainId={drain.id} />
                </div>
                <div className="relative z-20">
                  <DrainActions drain={{ id: drain.id, name: drain.name, status: drain.status }} />
                </div>
              </div>
            </ResourceListItem>
          );
        })}
      </ResourceListBody>
    </ResourceListContent>
  );
}
