import { Cube } from "@unkey/icons";

export const ResourceCardSkeleton = () => {
  return (
    <div className="p-5 flex flex-col border border-grayA-4 rounded-2xl w-full h-full gap-5">
      {/* Top Section */}
      <div className="flex gap-4 items-center min-h-11">
        <div className="size-10 bg-gray-3 rounded-[10px] flex items-center justify-center shrink-0 dark:ring-1 dark:ring-gray-4">
          <Cube iconSize="xl-medium" className="text-gray-11 opacity-30 shrink-0 size-5" />
        </div>
        <div className="flex flex-col w-full gap-2 py-[5px] min-w-0">
          <div className="h-[14px] w-24 bg-grayA-3 rounded-sm animate-pulse" />
          <div className="h-3 w-32 bg-grayA-3 rounded-sm animate-pulse" />
        </div>
      </div>

      {/* Middle Section */}
      <div className="flex flex-col gap-2">
        <div className="h-5 w-40 bg-grayA-3 rounded-sm animate-pulse" />

        <div className="flex gap-2 items-center min-w-0 justify-between min-h-5">
          <div className="h-4 w-16 bg-grayA-3 rounded-sm animate-pulse" />
          <div className="h-4 w-16 bg-grayA-3 rounded-sm animate-pulse" />
        </div>
      </div>
    </div>
  );
};
