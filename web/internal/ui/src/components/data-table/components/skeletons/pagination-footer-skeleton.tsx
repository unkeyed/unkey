export const PaginationFooterSkeleton = () => {
  return (
    <div className="w-[740px] border bg-gray-1 dark:bg-black border-gray-6 min-h-[60px] flex items-center justify-center rounded-[10px] drop-shadow-lg transform-gpu shadow-sm mb-5 pointer-events-none animate-fade-out">
      <div className="flex w-full justify-between items-center p-[18px]">
        {/* Item count skeleton */}
        <div className="flex items-center gap-2">
          <div className="h-3 w-12 bg-grayA-3 rounded animate-pulse" />
          <div className="h-3 w-8 bg-grayA-4 rounded animate-pulse" />
          <div className="h-3 w-4 bg-grayA-3 rounded animate-pulse" />
          <div className="h-3 w-6 bg-grayA-4 rounded animate-pulse" />
          <div className="h-3 w-14 bg-grayA-3 rounded animate-pulse" />
        </div>

        {/* Pagination controls skeleton */}
        <div className="flex items-center gap-1">
          {/* Prev button */}
          <div className="size-6 bg-grayA-3 rounded animate-pulse" />

          {/* Page pill group */}
          <div className="flex items-center bg-grayA-2 border border-grayA-3 rounded-lg p-0.5 gap-0.5">
            {Array.from({ length: 5 }).map((_, i) => (
              <div
                // biome-ignore lint/suspicious/noArrayIndexKey: static skeleton, order never changes
                key={i}
                className="w-7 h-7 bg-grayA-3 rounded-md animate-pulse"
                style={{ animationDelay: `${i * 60}ms` }}
              />
            ))}
          </div>

          {/* Next button */}
          <div className="size-6 bg-grayA-3 rounded animate-pulse" />

          {/* Minimize button */}
          <div className="size-6 bg-grayA-3 rounded ml-1 animate-pulse" />
        </div>
      </div>
    </div>
  );
};
