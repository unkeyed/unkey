import { Skeleton } from "@unkey/ui";
import { DEFAULT_NODE_WIDTH } from "../types";

export const SkeletonNode = () => {
  return (
    <div className={`relative rounded-2xl w-[${DEFAULT_NODE_WIDTH}px]`}>
      <div className="relative z-20 h-[100px] border rounded-2xl flex flex-col bg-raised shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)]">
        {/* Header */}
        <div
          className="border-b rounded-t-2xl px-3 py-2.5 flex"
          style={{
            background:
              "radial-gradient(circle at 5% 15%, var(--color-grayA-3) 0%, transparent 20%), light-dark(#FFF, #000)",
          }}
        >
          <div className="flex items-center gap-3">
            {/* Icon skeleton */}
            <Skeleton className="size-9 rounded-xl" />

            {/* Title/subtitle skeleton */}
            <div className="flex flex-col gap-2 justify-center h-9 py-2">
              <Skeleton className="h-1 w-16" />
              <Skeleton className="h-1 w-24" />
            </div>
          </div>

          {/* Status indicators skeleton */}
          <div className="flex gap-2 items-center ml-auto">
            <div className="w-[30px]" />
            <Skeleton className="w-[30px] h-[40px] rounded-lg" />
          </div>
        </div>

        {/* Footer */}
        <div className="px-1.5 py-1 flex items-center h-full bg-grayA-2 rounded-b-2xl">
          <Skeleton className="h-[22px] w-[63px] rounded-full bg-grayA-2" />
          <div className="flex items-center gap-2 ml-auto">
            <Skeleton className="h-5 w-14 rounded-full bg-grayA-2" />
            <Skeleton className="h-5 w-14 rounded-full bg-grayA-2" />
          </div>
        </div>
      </div>
    </div>
  );
};
