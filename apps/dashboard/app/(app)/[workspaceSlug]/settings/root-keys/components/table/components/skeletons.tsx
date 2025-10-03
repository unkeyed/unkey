import { cn } from "@/lib/utils";
import { ChartActivity2, Dots, Key2, Page2 } from "@unkey/icons";

export const RootKeyColumnSkeleton = () => (
  <div className="flex flex-col items-start px-[18px] py-[6px]">
    <div className="flex gap-4 items-center">
      <div className="size-5 rounded flex items-center justify-center border border-grayA-3 bg-grayA-3 animate-pulse">
        <Key2 iconsize="sm-regular" className="text-gray-12 opacity-50" />
      </div>
      <div className="h-4 w-24 bg-grayA-3 rounded animate-pulse" />
    </div>
  </div>
);

export const CreatedAtColumnSkeleton = () => (
  <div className="flex flex-col items-start py-[6px]">
    <div className="h-4 w-24 bg-grayA-3 rounded animate-pulse" />
  </div>
);
export const KeyColumnSkeleton = () => (
  <div className="rounded-lg border bg-grayA-2 border-grayA-3 text-transparent w-[160px] px-2 py-1 flex gap-2 items-center h-[28px] animate-pulse">
    <div className="h-2 w-2 bg-grayA-3 rounded-full animate-pulse" />
    <div className="h-2 w-full bg-grayA-3 rounded animate-pulse" />
  </div>
);

export const PermissionsColumnSkeleton = () => (
  <div className="flex flex-col gap-1 py-2 max-w-[200px]">
    <div className="rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 border border-dashed bg-grayA-3 border-grayA-6 animate-pulse h-[22px]">
      <Page2 className="size-3 opacity-50" iconsize="md-medium" />
      <div className="h-2 w-20 bg-grayA-3 rounded animate-pulse" />
    </div>
  </div>
);

export const LastUpdatedColumnSkeleton = () => (
  <div className="px-1.5 rounded-md flex gap-2 items-center max-w-min h-[22px] bg-grayA-3 border-none animate-pulse">
    <ChartActivity2 iconsize="sm-regular" className="opacity-50" />
    <div className="h-2 w-16 bg-grayA-3 rounded animate-pulse" />
  </div>
);

export const ActionColumnSkeleton = () => (
  <button
    type="button"
    className={cn(
      "group size-5 p-0 rounded m-0 items-center flex justify-center animate-pulse",
      "border border-gray-6",
    )}
  >
    <Dots className="text-gray-11" iconsize="sm-regular" />
  </button>
);
