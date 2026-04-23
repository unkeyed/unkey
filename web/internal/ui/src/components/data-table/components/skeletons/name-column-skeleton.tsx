import type { ReactNode } from "react";

export interface NameColumnSkeletonProps {
  icon: ReactNode;
  lines?: 1 | 2;
}

export const NameColumnSkeleton = ({ icon, lines = 2 }: NameColumnSkeletonProps) => (
  <div className="flex flex-col items-start px-[18px] py-[6px]">
    <div className="flex gap-4 items-center">
      <div className="size-5 rounded-sm flex items-center justify-center border border-grayA-3 bg-grayA-3 animate-pulse">
        {icon}
      </div>
      {lines === 2 ? (
        <div className="flex flex-col gap-1">
          <div className="h-4 w-24 bg-grayA-3 rounded-sm animate-pulse" />
          <div className="h-4 w-32 bg-grayA-3 rounded-sm animate-pulse" />
        </div>
      ) : (
        <div className="h-4 w-24 bg-grayA-3 rounded-sm animate-pulse" />
      )}
    </div>
  </div>
);
