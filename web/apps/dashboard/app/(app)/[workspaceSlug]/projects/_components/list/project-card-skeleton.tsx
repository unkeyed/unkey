export function ProjectCardSkeleton() {
  return (
    <div className="p-5 flex flex-col justify-between border border-grayA-4 rounded-2xl w-full h-full gap-6">
      <div className="flex items-start justify-between min-h-5">
        <div className="h-[14px] w-28 bg-grayA-3 rounded-sm animate-pulse" />
        <div className="size-5 bg-grayA-3 rounded-sm animate-pulse" />
      </div>

      <div className="flex items-center">
        {Array.from({ length: 3 }).map((_, i) => (
          <div
            // biome-ignore lint/suspicious/noArrayIndexKey: skeleton items don't need stable keys
            key={i}
            className="size-7 rounded-full bg-grayA-3 ring-2 ring-gray-1 animate-pulse -ml-2 first:ml-0"
          />
        ))}
      </div>
    </div>
  );
}
