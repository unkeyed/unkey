import { ChartActivity2, Dots, Key2, Page2 } from "@unkey/icons";
import { cn } from "../../../../lib/utils";
import { DashedBadgeSkeleton } from "./dashed-badge-skeleton";
import { NameColumnSkeleton } from "./name-column-skeleton";

export const RootKeyColumnSkeleton = () => (
  <NameColumnSkeleton
    icon={<Key2 iconSize="sm-regular" className="text-gray-12 opacity-50" />}
    lines={1}
  />
);

export const CreatedAtColumnSkeleton = () => (
  <div className="flex flex-col items-start">
    <div className="h-4 w-24 bg-grayA-3 rounded-sm animate-pulse" />
  </div>
);
export const KeyColumnSkeleton = () => (
  <div className="rounded-lg border bg-grayA-2 border-grayA-3 text-transparent w-[160px] px-2 py-1 flex gap-2 items-center h-[28px] animate-pulse">
    <div className="h-2 w-2 bg-grayA-3 rounded-full animate-pulse" />
    <div className="h-2 w-full bg-grayA-3 rounded-sm animate-pulse" />
  </div>
);

export const PermissionsColumnSkeleton = () => (
  <DashedBadgeSkeleton icon={<Page2 iconSize="md-medium" className="opacity-50" />} />
);

export const LastUpdatedColumnSkeleton = () => (
  <div className="px-1.5 rounded-md flex gap-2 items-center max-w-min h-[22px] bg-grayA-3 border-none animate-pulse">
    <ChartActivity2 iconSize="sm-regular" className="opacity-50" />
    <div className="h-2 w-16 bg-grayA-3 rounded-sm animate-pulse" />
  </div>
);

export const ActionColumnSkeleton = () => (
  <button
    type="button"
    className={cn(
      "group size-5 p-0 rounded-sm m-0 items-center flex justify-center animate-pulse",
      "border border-gray-6",
    )}
  >
    <Dots className="text-gray-11" iconSize="sm-regular" />
  </button>
);
