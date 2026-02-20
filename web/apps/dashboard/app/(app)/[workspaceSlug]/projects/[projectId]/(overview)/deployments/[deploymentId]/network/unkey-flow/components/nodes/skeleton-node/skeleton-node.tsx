import { DEFAULT_NODE_WIDTH } from "../types";

export const SkeletonNode = () => {
  return (
    <div className={`relative rounded-[14px] w-[${DEFAULT_NODE_WIDTH}px]`}>
      <div className="relative z-20 h-[100px] border border-grayA-4 rounded-[14px] flex flex-col bg-white dark:bg-black shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)]">
        {/* Header */}
        <div
          className="border-b border-grayA-4 rounded-t-[14px] px-3 py-2.5 flex"
          style={{
            background:
              "radial-gradient(circle at 5% 15%, hsl(var(--grayA-3)) 0%, transparent 20%), light-dark(#FFF, #000)",
          }}
        >
          <div className="flex items-center gap-3">
            {/* Icon skeleton */}
            <div className="size-9 rounded-[10px] bg-grayA-3 animate-pulse" />

            {/* Title/subtitle skeleton */}
            <div className="flex flex-col gap-2 justify-center h-9 py-2">
              <div className="h-1 w-16 bg-grayA-3 rounded-sm animate-pulse" />
              <div className="h-1 w-24 bg-grayA-3 rounded-sm animate-pulse" />
            </div>
          </div>

          {/* Status indicators skeleton */}
          <div className="flex gap-2 items-center ml-auto">
            <div className="w-[30px] h-[40px] rounded-lg bg-grayA-3 animate-pulse" />
            <div className="w-[30px] h-[40px] rounded-lg bg-grayA-3 animate-pulse" />
          </div>
        </div>

        {/* Footer */}
        <div className="px-1.5 py-1 flex items-center h-full bg-grayA-2 rounded-b-[14px]">
          <div className="h-[22px] w-[63px] rounded-full bg-grayA-2 animate-pulse" />
          <div className="flex items-center gap-2 ml-auto">
            <div className="h-5 w-14 rounded-full bg-grayA-2 animate-pulse" />
            <div className="h-5 w-14 rounded-full bg-grayA-2 animate-pulse" />
          </div>
        </div>
      </div>
    </div>
  );
};
