export function RegionFlagsSkeleton() {
  return (
    <div className="gap-1 flex items-center justify-center border border-grayA-3 transition-all duration-100 bg-grayA-3 p-1.5 h-[22px] rounded-md">
      {Array.from({ length: 3 }).map((_, idx) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: skeleton items don't need stable keys
        <div key={`${idx}-skeleton`} className="size-4 bg-grayA-4 rounded-[10px] animate-pulse" />
      ))}
    </div>
  );
}
