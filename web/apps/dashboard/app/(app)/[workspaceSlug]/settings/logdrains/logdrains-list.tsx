"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { Earth } from "@unkey/icons";
import { Empty, ResourceListBody, ResourceListContent, ResourceListItem } from "@unkey/ui";
import Link from "next/link";
import { AxiomLogo } from "./axiom-logo";
import { CreateLogdrainButton } from "./create-logdrain-button";
import { DrainRowChart } from "./drain-row-chart";
import { DrainActions, StatusBadge } from "./logdrain-ui";

const SKELETON_ROWS = 8;

function LogdrainsSkeleton() {
  return (
    <ResourceListContent aria-busy="true">
      <output className="sr-only">Loading log drains...</output>
      <ResourceListBody aria-hidden="true">
        {Array.from({ length: SKELETON_ROWS }).map((_, index) => (
          <ResourceListItem
            // biome-ignore lint/suspicious/noArrayIndexKey: skeleton rows are static and never reorder
            key={index}
            className="flex flex-col gap-3 px-4 py-3 md:flex-row md:items-center md:gap-0"
          >
            <div className="flex min-w-0 flex-col gap-1 md:w-[30%] md:shrink-0">
              <div className="h-3.5 w-40 animate-pulse rounded-sm bg-grayA-3" />
              <div className="h-3 w-24 animate-pulse rounded-sm bg-grayA-3" />
            </div>
            <div className="flex items-center gap-2 md:w-[20%] md:shrink-0">
              <div className="size-5 animate-pulse rounded-sm bg-grayA-3" />
              <div className="h-3 w-12 animate-pulse rounded-sm bg-grayA-3" />
            </div>
            <div className="md:w-[20%] md:shrink-0">
              <div className="h-5 w-20 animate-pulse rounded-md bg-grayA-3" />
            </div>
            <div className="flex items-center gap-3 md:w-[30%] md:shrink-0 md:justify-end">
              <div className="h-7 w-[158px] animate-pulse rounded-md bg-grayA-3" />
              <div className="size-5 animate-pulse rounded-sm bg-grayA-3" />
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
      <ResourceListContent>
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
      <ResourceListContent>
        <div className="p-3">
          <Empty className="min-h-[200px] rounded-lg border border-dashed border-gray-4 bg-gray-1/50">
            <Empty.Title>No log drains</Empty.Title>
            <Empty.Description className="max-w-sm">
              Stream audit logs to HTTP or Axiom.
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
    <ResourceListContent>
      <ResourceListBody aria-label="Log drains">
        {query.data.map((drain) => {
          const targetLabel = drain.kind === "http" ? "HTTP" : "Axiom";
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
              <div className="flex items-center gap-2 md:w-[20%] md:shrink-0">
                <span className="flex size-5 shrink-0 items-center justify-center rounded-sm bg-grayA-3 text-gray-11">
                  {drain.kind === "http" ? (
                    <Earth iconSize="sm-regular" />
                  ) : (
                    <AxiomLogo className="size-3.5" />
                  )}
                </span>
                <span className="text-[13px] font-medium text-accent-12">{targetLabel}</span>
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
