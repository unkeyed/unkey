import { IconCubeOutline18 } from "@unkey/icons";
import { Skeleton } from "@unkey/ui";

export const ResourceCardSkeleton = () => {
  return (
    <div className="p-5 flex flex-col border bg-raised shadow-xs rounded-lg w-full h-full gap-5">
      {/* Top Section */}
      <div className="flex gap-4 items-center min-h-11">
        <div className="size-10 bg-gray-3 rounded-xl flex items-center justify-center shrink-0 dark:ring-1 dark:ring-gray-4">
          <IconCubeOutline18 className="text-gray-11 opacity-30 shrink-0 size-5" />
        </div>
        <div className="flex flex-col w-full gap-2 py-[5px] min-w-0">
          <Skeleton className="h-[14px] w-24" />
          <Skeleton className="h-3 w-32" />
        </div>
      </div>

      {/* Middle Section */}
      <div className="flex flex-col gap-2">
        <Skeleton className="h-5 w-40" />

        <div className="flex gap-2 items-center min-w-0 justify-between min-h-5">
          <Skeleton className="h-4 w-16" />
          <Skeleton className="h-4 w-16" />
        </div>
      </div>
    </div>
  );
};
