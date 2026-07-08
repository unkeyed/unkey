export function ApiCardSkeleton() {
  return (
    <div className="relative h-full p-5 flex flex-col border border-grayA-4 rounded-lg w-full gap-5 min-h-[152px]">
      <div className="flex flex-col w-full gap-2 min-w-0">
        <div className="h-[14px] w-32 bg-grayA-3 rounded-sm animate-pulse" />
        <div className="h-3 w-40 bg-grayA-3 rounded-sm animate-pulse" />
      </div>
      <div className="mt-auto flex flex-col gap-3">
        <div className="h-7 w-full bg-grayA-2 rounded-sm animate-pulse" />
        <div className="flex gap-2 items-center min-w-0 h-3">
          <div className="h-3 w-20 bg-grayA-3 rounded-sm animate-pulse" />
          <span className="text-gray-7">·</span>
          <div className="h-3 w-28 bg-grayA-3 rounded-sm animate-pulse" />
        </div>
      </div>
    </div>
  );
}
