import { Skeleton } from "@unkey/ui";
export const TimeColumnSkeleton = () => <Skeleton className="my-2 h-3 w-[75px] rounded" />;

export const SeverityColumnSkeleton = () => <Skeleton className="my-2 mx-1 size-4 rounded-full" />;

export const RegionColumnSkeleton = () => (
  <div className="my-2 flex items-center gap-1.5">
    <Skeleton className="size-4 rounded-full" />
    <Skeleton className="h-3 w-20 rounded" />
  </div>
);

export const MessageColumnSkeleton = () => <Skeleton className="my-2 h-3 w-125 rounded" />;
