import type { ReactNode } from "react";
import { cn } from "../../../../lib/utils";

export interface DashedBadgeSkeletonProps {
  icon: ReactNode;
  barWidthClass?: string;
  className?: string;
}

export const DashedBadgeSkeleton = ({
  icon,
  barWidthClass = "w-16",
  className,
}: DashedBadgeSkeletonProps) => (
  <div className={cn("flex flex-col gap-1 py-2 max-w-[200px]", className)}>
    <div className="rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 border border-dashed bg-grayA-3 border-grayA-6 animate-pulse h-[22px]">
      {icon}
      <div className={cn("h-2 bg-grayA-3 rounded-sm animate-pulse", barWidthClass)} />
    </div>
  </div>
);
