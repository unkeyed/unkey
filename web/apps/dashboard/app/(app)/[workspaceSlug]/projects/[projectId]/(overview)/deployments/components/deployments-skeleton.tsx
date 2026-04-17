"use client";

import { Dots } from "@unkey/icons";

export function DeploymentsSkeleton() {
  return (
    <div className="border border-grayA-4 rounded-[14px] overflow-hidden divide-y divide-grayA-4">
      {Array.from({ length: 8 }).map((_, i) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: skeleton items don't need stable keys
        <div key={i} className="flex flex-col md:flex-row md:items-center px-4 py-3 gap-3 md:gap-0">
          {/* Identity + Status */}
          <div className="flex items-center justify-between md:contents">
            <div className="md:w-[25%] md:shrink-0 flex flex-col gap-2 min-w-0">
              <div className="h-[14px] w-20 bg-grayA-3 rounded-sm animate-pulse" />
              <div className="h-3 w-16 bg-grayA-3 rounded-sm animate-pulse" />
            </div>
            <div className="md:w-[15%] md:shrink-0">
              <div className="h-5.5 w-20 bg-grayA-3 rounded-md animate-pulse" />
            </div>
          </div>

          {/* Source */}
          <div className="md:w-[30%] md:shrink-0 flex flex-col gap-2 min-w-0">
            <div className="flex items-center gap-2">
              <div className="size-4 bg-grayA-3 rounded animate-pulse shrink-0" />
              <div className="h-[14px] w-20 bg-grayA-3 rounded-sm animate-pulse" />
              <div className="h-[14px] w-14 bg-grayA-3 rounded-sm animate-pulse" />
            </div>
            <div className="pl-6">
              <div className="h-3 w-32 bg-grayA-3 rounded-sm animate-pulse" />
            </div>
          </div>

          {/* Meta */}
          <div className="md:w-[30%] md:shrink-0 flex items-center md:justify-end gap-3">
            <div className="h-[14px] w-12 bg-grayA-3 rounded-sm animate-pulse" />
            <div className="size-5 bg-grayA-3 rounded-full animate-pulse" />
            <Dots iconSize="sm-regular" className="text-gray-11 opacity-50" />
          </div>
        </div>
      ))}
    </div>
  );
}
