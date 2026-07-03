"use client";

export function ScheduledDeletionsSkeleton() {
  return (
    <div className="border border-grayA-4 rounded-[14px] overflow-hidden divide-y divide-grayA-4">
      {Array.from({ length: 4 }).map((_, i) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: skeleton items don't need stable keys
        <div key={i} className="flex flex-col md:flex-row md:items-center px-4 py-3 gap-3 md:gap-0">
          <div className="flex items-center justify-between md:contents">
            <div className="md:w-[35%] md:shrink-0 flex flex-col gap-1 min-w-0">
              <div className="h-[14px] w-32 bg-grayA-3 rounded-sm animate-pulse" />
              <div className="h-3 w-16 bg-grayA-3 rounded-sm animate-pulse" />
            </div>
            <div className="md:w-[35%] md:shrink-0">
              <div className="h-[14px] w-24 bg-grayA-3 rounded-sm animate-pulse" />
            </div>
          </div>
          <div className="md:w-[30%] md:shrink-0 flex items-center md:justify-end">
            <div className="h-8 w-20 bg-grayA-3 rounded-md animate-pulse" />
          </div>
        </div>
      ))}
    </div>
  );
}
