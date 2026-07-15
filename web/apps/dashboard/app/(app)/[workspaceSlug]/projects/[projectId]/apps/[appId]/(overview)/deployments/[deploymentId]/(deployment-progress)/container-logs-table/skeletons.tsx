export const TimeColumnSkeleton = () => (
  <div className="my-2 h-3 w-[75px] bg-grayA-3 rounded animate-pulse" />
);

export const SeverityColumnSkeleton = () => (
  <div className="my-2 mx-1 size-4 bg-grayA-3 rounded-full animate-pulse" />
);

export const RegionColumnSkeleton = () => (
  <div className="my-2 flex items-center gap-1.5">
    <div className="size-4 bg-grayA-3 rounded-full animate-pulse" />
    <div className="h-3 w-20 bg-grayA-3 rounded animate-pulse" />
  </div>
);

export const MessageColumnSkeleton = () => (
  <div className="my-2 h-3 w-125 bg-grayA-3 rounded animate-pulse" />
);
