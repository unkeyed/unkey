"use client";

import { Dots } from "@unkey/icons";
import { ResourceListBody, ResourceListContent, ResourceListItem } from "@unkey/ui";

export function DeploymentsSkeleton({ rows = 8 }: { rows?: number }) {
  return (
    <ResourceListContent aria-busy="true">
      <output className="sr-only">Loading deployments...</output>
      <ResourceListBody aria-hidden="true">
        {Array.from({ length: rows }).map((_, index) => (
          <ResourceListItem
            // biome-ignore lint/suspicious/noArrayIndexKey: skeleton rows are static and never reorder
            key={index}
            className="flex items-center gap-3 px-4 py-2.5"
          >
            <div className="flex min-w-0 flex-1 items-center">
              <div className="h-[14px] w-48 animate-pulse rounded-sm bg-grayA-3" />
            </div>
            <div className="flex w-44 shrink-0 items-center gap-2">
              <div className="size-2 animate-pulse rounded-full bg-grayA-3" />
              <div className="h-[14px] w-16 animate-pulse rounded-sm bg-grayA-3" />
            </div>
            <div className="flex w-32 shrink-0 items-center">
              <div className="h-5.5 w-24 animate-pulse rounded-md bg-grayA-3" />
            </div>
            <div className="hidden w-32 shrink-0 items-center gap-1.5 md:flex">
              <div className="size-4 animate-pulse rounded bg-grayA-3" />
              <div className="h-[14px] w-16 animate-pulse rounded-sm bg-grayA-3" />
            </div>
            <div className="hidden w-40 shrink-0 items-center gap-2 lg:flex">
              <div className="size-4 animate-pulse rounded bg-grayA-3" />
              <div className="h-[14px] w-28 animate-pulse rounded-sm bg-grayA-3" />
            </div>
            <div className="ml-auto flex shrink-0 items-center gap-3">
              <div className="flex w-36 justify-end">
                <div className="h-[14px] w-16 animate-pulse rounded-sm bg-grayA-3" />
              </div>
              <div className="hidden size-5 animate-pulse rounded-full bg-grayA-3 md:block" />
              <Dots iconSize="sm-regular" className="text-gray-11 opacity-50" />
            </div>
          </ResourceListItem>
        ))}
      </ResourceListBody>
    </ResourceListContent>
  );
}
