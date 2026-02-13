import { Dots } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";

export const IdentityColumnSkeleton = () => (
  <div className="flex flex-col items-start w-auto">
    <div className="flex gap-4 items-center">
      <div className="bg-grayA-3 size-5 rounded-sm animate-pulse" />
      <div className="flex flex-col gap-1">
        <div className="h-2 w-40 bg-grayA-3 rounded-sm animate-pulse" />
        <div className="h-2 w-16 bg-grayA-3 rounded-sm animate-pulse mt-1" />
      </div>
    </div>
  </div>
);

export const ExternalIdColumnSkeleton = () => (
  <div className="h-2 w-32 bg-grayA-3 rounded-sm animate-pulse" />
);

export const CountColumnSkeleton = () => (
  <div className="h-2 w-8 bg-grayA-3 rounded-sm animate-pulse" />
);

export const CreatedColumnSkeleton = () => (
  <div className="h-2 w-24 bg-grayA-3 rounded-sm animate-pulse" />
);

export const LastUsedColumnSkeleton = () => (
  <div className="px-1.5 rounded-md flex gap-2 items-center w-[140px] h-[22px] bg-grayA-3 animate-pulse">
    <div className="h-2 w-2 bg-grayA-3 rounded-full animate-pulse" />
    <div className="h-2 w-12 bg-grayA-3 rounded-sm animate-pulse" />
    <div className="h-2 w-12 bg-grayA-3 rounded-sm animate-pulse" />
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
