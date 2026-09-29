import { IconDotsOutline12 } from "@unkey/icons";
import { Skeleton } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";

export const KeyColumnSkeleton = () => (
  <div className="flex flex-col items-start w-auto">
    <div className="flex gap-4 items-center">
      <Skeleton className="size-5" />
      <div className="flex flex-col gap-1">
        <Skeleton className="h-2 w-40" />
        <Skeleton className="h-2 w-16 mt-1" />
      </div>
    </div>
  </div>
);

export const ValueColumnSkeleton = () => (
  <div className="rounded-lg border bg-grayA-2 text-transparent w-[160px] px-2 py-1 flex gap-2 items-center h-[28px] animate-pulse">
    <Skeleton className="h-2 w-2 rounded-full" />
    <Skeleton className="h-2 w-full" />
  </div>
);

export const UsageColumnSkeleton = ({ maxBars = 30 }: { maxBars?: number }) => (
  <div
    className={cn(
      "grid items-end h-[28px] bg-grayA-2 w-[158px] border border-transparent px-1 py-0 overflow-hidden rounded-md",
      "animate-pulse",
    )}
    style={{
      gridTemplateColumns: `repeat(${maxBars}, 3px)`,
      gap: "2px",
    }}
  >
    {Array(maxBars)
      .fill(0)
      .map((_, index) => (
        <div
          key={`loading-${
            // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
            index
          }`}
          className="flex flex-col"
        >
          <Skeleton
            className="w-[3px] bg-grayA-5"
            style={{ height: `${2 + Math.floor(Math.random() * 20)}px` }}
          />
        </div>
      ))}
  </div>
);

export const LastUsedColumnSkeleton = () => (
  <div className="px-1.5 rounded-md flex gap-2 items-center w-35 h-5.5 bg-grayA-3 animate-pulse">
    <Skeleton className="h-2 w-2 rounded-full" />
    <Skeleton className="h-2 w-12" />
    <Skeleton className="h-2 w-12" />
  </div>
);

export const StatusColumnSkeleton = () => (
  <div
    className="flex w-25 items-center h-5.5 gap-2 px-1.5 py-1 rounded-md bg-grayA-3 animate-pulse"
    aria-busy="true"
    aria-live="polite"
  >
    <Skeleton className="h-2 w-2 rounded-full" />
    <Skeleton className="h-2 w-16" />
  </div>
);

export const ActionColumnSkeleton = () => (
  <button
    type="button"
    className={cn(
      "group size-5 p-0 rounded-sm m-0 items-center flex justify-center animate-pulse",
      "border",
    )}
  >
    <IconDotsOutline12 className="text-gray-11" />
  </button>
);
