import { Dots } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";

export const KeyColumnSkeleton = () => (
  <div className="flex flex-col items-start w-auto">
    <div className="flex gap-4 items-center">
      <div className="bg-grayA-3 size-5 rounded animate-pulse" />
      <div className="flex flex-col gap-1">
        <div className="h-2 w-40 bg-grayA-3 rounded animate-pulse" />
        <div className="h-2 w-16 bg-grayA-3 rounded animate-pulse mt-1" />
      </div>
    </div>
  </div>
);

export const ValueColumnSkeleton = () => (
  <div className="rounded-lg border bg-grayA-2 border-grayA-3 text-transparent w-[160px] px-2 py-1 flex gap-2 items-center h-[28px] animate-pulse">
    <div className="h-2 w-2 bg-grayA-3 rounded-full animate-pulse" />
    <div className="h-2 w-full bg-grayA-3 rounded animate-pulse" />
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
          <div
            className="w-[3px] bg-grayA-5 animate-pulse"
            style={{ height: `${2 + Math.floor(Math.random() * 20)}px` }}
          />
        </div>
      ))}
  </div>
);

export const LastUsedColumnSkeleton = () => (
  <div className="px-1.5 rounded-md flex gap-2 items-center w-[140px] h-[22px] bg-grayA-3 animate-pulse">
    <div className="h-2 w-2 bg-grayA-3 rounded-full animate-pulse" />
    <div className="h-2 w-12 bg-grayA-3 rounded animate-pulse" />
    <div className="h-2 w-12 bg-grayA-3 rounded animate-pulse" />
  </div>
);

export const StatusColumnSkeleton = () => (
  <div
    className="flex w-[100px] items-center h-[22px] space-x-2 px-1.5 py-1 rounded-md bg-grayA-3 animate-pulse"
    aria-busy="true"
    aria-live="polite"
  >
    <div className="h-2 w-2 bg-grayA-3 rounded-full animate-pulse" />
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
    <Dots className="text-gray-11" size="sm-regular" />
  </button>
);
