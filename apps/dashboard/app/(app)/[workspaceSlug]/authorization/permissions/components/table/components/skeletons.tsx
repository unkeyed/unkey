import { cn } from "@/lib/utils";
import { ChartActivity2, Dots, Key2, Page2, Tag } from "@unkey/icons";

export const RoleColumnSkeleton = () => (
  <div className="flex flex-col items-start px-[18px] py-[6px]">
    <div className="flex gap-4 items-center">
      <div className="size-5 rounded flex items-center justify-center border border-grayA-3 bg-grayA-3 animate-pulse">
        <Tag iconSize="sm-regular" className="text-gray-12 opacity-50" />
      </div>
      <div className="flex flex-col gap-1">
        <div className="h-4 w-24 bg-grayA-3 rounded animate-pulse" />
        <div className="h-4 w-32 bg-grayA-3 rounded animate-pulse" />
      </div>
    </div>
  </div>
);

export const SlugColumnSkeleton = () => (
  <div className="flex flex-col gap-1 py-2 max-w-[200px]">
    <div className="rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 border border-dashed bg-grayA-3 border-grayA-6 animate-pulse h-[22px]">
      <Page2 iconSize="md-medium" className="opacity-50" />
      <div className="h-2 w-16 bg-grayA-3 rounded animate-pulse" />
    </div>
  </div>
);

export const AssignedKeysColumnSkeleton = () => (
  <div className="flex flex-col gap-1 py-2 max-w-[200px]">
    <div className="rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 border border-dashed bg-grayA-3 border-grayA-6 animate-pulse h-[22px]">
      <Tag iconSize="md-medium" className="opacity-50" />
      <div className="h-2 w-16 bg-grayA-3 rounded animate-pulse" />
    </div>
  </div>
);

export const AssignedToKeysColumnSkeleton = () => (
  <div className="flex flex-col gap-1 py-2 max-w-[200px]">
    <div className="rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 border border-dashed bg-grayA-3 border-grayA-6 animate-pulse h-[22px]">
      <Key2 iconSize="md-medium" className="opacity-50" />
      <div className="h-2 w-16 bg-grayA-3 rounded animate-pulse" />
    </div>
  </div>
);

export const LastUpdatedColumnSkeleton = () => (
  <div className="px-1.5 rounded-md flex gap-2 items-center max-w-min h-[22px] bg-grayA-3 border-none animate-pulse">
    <ChartActivity2 iconSize="sm-regular" className="opacity-50" />
    <div className="h-2 w-16 bg-grayA-3 rounded animate-pulse" />
  </div>
);

export const ActionColumnSkeleton = () => (
  <button
    type="button"
    className={cn(
      "group size-5 p-0 rounded m-0 items-center flex justify-center animate-pulse",
      "border border-gray-6"
    )}
  >
    <Dots className="text-gray-11" iconSize="sm-regular" />
  </button>
);
