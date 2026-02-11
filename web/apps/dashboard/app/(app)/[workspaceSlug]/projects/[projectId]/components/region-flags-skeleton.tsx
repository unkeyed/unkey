export function RegionFlagsSkeleton() {
  return (
    <div className="gap-1 flex items-center justify-center border border-grayA-3 transition-all duration-100 bg-grayA-3 p-1.5 h-[22px] rounded-md">
      {Array.from({ length: 3 }).map((_, i) => (
        <div
          key={i}
          className="size-4 bg-grayA-4 rounded-[10px] animate-pulse"
        />
      ))}
    </div>
  );
}
