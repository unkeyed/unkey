export function ProjectCardSkeleton() {
  return (
    // min-h matches a populated ProjectCard (~120.5px) so the grid doesn't shift on load.
    <div className="p-5 flex flex-col justify-between border border-grayA-4 rounded-2xl w-full h-full min-h-[120px] gap-6">
      {/* Mirrors ProjectCard's header: 20px title line beside a size-6 actions button. */}
      <div className="flex gap-4 items-start justify-between min-h-5">
        <div className="h-[14px] w-28 bg-grayA-3 rounded-sm animate-pulse" />
        <div className="size-6 bg-grayA-3 rounded-md animate-pulse shrink-0" />
      </div>

      {/* Single size-7 blob to match the app icon stack's row height. */}
      <div className="flex items-center">
        <div className="size-7 rounded-full bg-grayA-3 animate-pulse" />
      </div>
    </div>
  );
}
