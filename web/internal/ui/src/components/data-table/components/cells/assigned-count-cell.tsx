import type { ReactNode } from "react";
import { cn } from "../../../../lib/utils";

export type AssignedCountCellProps = {
  count: number;
  icon: ReactNode;
  singularLabel: string;
  isSelected?: boolean;
};

export const AssignedCountCell = ({
  count,
  icon,
  singularLabel,
  isSelected = false,
}: AssignedCountCellProps) => {
  if (count === 0) {
    return (
      <div className="flex flex-col gap-1 py-2 max-w-[200px]">
        <div
          className={cn(
            "rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed bg-grayA-2",
            isSelected ? "border-grayA-7 text-grayA-9" : "border-grayA-6 text-grayA-8",
          )}
        >
          {icon}
          <span className="text-grayA-9 text-xs">None assigned</span>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1 py-2 max-w-[200px]">
      <div
        className={cn(
          "font-mono rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed text-grayA-12",
          isSelected
            ? "bg-grayA-4 border-grayA-7"
            : "bg-grayA-3 border-grayA-6 group-hover:bg-grayA-4",
        )}
      >
        {icon}
        <div className="text-grayA-11 text-xs max-w-[150px] truncate">
          {count} {count === 1 ? singularLabel : `${singularLabel}s`}
        </div>
      </div>
    </div>
  );
};
