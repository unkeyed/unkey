import { cn } from "@unkey/ui/src/lib/utils";

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
