"use client";

import { Dots } from "@unkey/icons";
import { ResourceListBody, ResourceListContent, ResourceListItem } from "@unkey/ui";

const SKELETON_ROWS = [
  "deployment-1",
  "deployment-2",
  "deployment-3",
  "deployment-4",
  "deployment-5",
  "deployment-6",
  "deployment-7",
  "deployment-8",
];

export function DeploymentsSkeleton() {
  return (
    <ResourceListContent aria-busy="true">
      <output className="sr-only">Loading deployments...</output>
      <ResourceListBody aria-hidden="true">
        {SKELETON_ROWS.map((row) => (
          <ResourceListItem
            key={row}
            className="flex flex-col gap-3 px-4 py-3 md:flex-row md:items-center md:gap-0"
          >
            {/* Identity + Status */}
            <div className="flex items-center justify-between md:contents">
              <div className="md:w-[20%] md:shrink-0 flex flex-col gap-1 min-w-0">
                <div className="h-[14px] w-20 bg-grayA-3 rounded-sm animate-pulse" />
                <div className="h-3 w-16 bg-grayA-3 rounded-sm animate-pulse" />
              </div>
              <div className="md:w-[20%] md:shrink-0">
                <div className="h-5.5 w-20 bg-grayA-3 rounded-md animate-pulse" />
              </div>
            </div>

            {/* Source */}
            <div className="md:w-[30%] md:shrink-0 flex flex-col gap-1 min-w-0">
              <div className="flex items-center gap-2 min-w-0">
                <div className="size-4 bg-grayA-3 rounded animate-pulse shrink-0" />
                <div className="h-[14px] w-20 bg-grayA-3 rounded-sm animate-pulse" />
                <div className="h-[14px] w-14 bg-grayA-3 rounded-sm animate-pulse" />
              </div>
              <div className="flex items-center gap-2 min-w-0">
                <div className="size-4 bg-grayA-3 rounded animate-pulse shrink-0" />
                <div className="h-3 w-32 bg-grayA-3 rounded-sm animate-pulse" />
              </div>
            </div>

            {/* Meta */}
            <div className="md:w-[30%] md:shrink-0 flex items-center md:justify-end gap-3">
              <div className="h-[14px] w-12 bg-grayA-3 rounded-sm animate-pulse" />
              <div className="size-5 bg-grayA-3 rounded-full animate-pulse" />
              <Dots iconSize="sm-regular" className="text-gray-11 opacity-50" />
            </div>
          </ResourceListItem>
        ))}
      </ResourceListBody>
    </ResourceListContent>
  );
}
