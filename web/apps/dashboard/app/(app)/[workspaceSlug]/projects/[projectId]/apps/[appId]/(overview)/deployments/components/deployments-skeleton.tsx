"use client";

import { IconDotsOutline12 } from "@unkey/icons";
import { ResourceListBody, ResourceListContent, ResourceListItem, Skeleton } from "@unkey/ui";

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
              <Skeleton className="h-[14px] w-48" />
            </div>
            <div className="flex w-44 shrink-0 items-center gap-2">
              <Skeleton className="size-2 rounded-full" />
              <Skeleton className="h-[14px] w-16" />
            </div>
            <div className="flex w-32 shrink-0 items-center">
              <Skeleton className="h-5.5 w-24 rounded-md" />
            </div>
            <div className="hidden w-32 shrink-0 items-center gap-1.5 md:flex">
              <Skeleton className="size-4 rounded" />
              <Skeleton className="h-[14px] w-16" />
            </div>
            <div className="hidden w-40 shrink-0 items-center gap-2 lg:flex">
              <Skeleton className="size-4 rounded" />
              <Skeleton className="h-[14px] w-28" />
            </div>
            <div className="ml-auto flex shrink-0 items-center gap-3">
              <div className="flex w-36 justify-end">
                <Skeleton className="h-[14px] w-16" />
              </div>
              <Skeleton className="hidden size-5 rounded-full md:block" />
              <IconDotsOutline12 className="text-gray-11 opacity-50" />
            </div>
          </ResourceListItem>
        ))}
      </ResourceListBody>
    </ResourceListContent>
  );
}
