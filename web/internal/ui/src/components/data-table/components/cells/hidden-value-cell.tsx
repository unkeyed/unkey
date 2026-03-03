import { CircleLock } from "@unkey/icons";
import type React from "react";
import { cn } from "../../../../lib/utils";
import { toast } from "../../../toaster";

export interface HiddenValueCellProps {
  value: string;
  title: string;
  selected: boolean;
}

export const HiddenValueCell = ({ value, title = "Value", selected }: HiddenValueCellProps) => {
  // Show only first 4 characters, then dots
  const displayValue = value.padEnd(16, "•");

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    navigator.clipboard
      .writeText(value)
      .then(() => {
        toast.success(`${title} copied to clipboard`);
      })
      .catch((error) => {
        console.error("Failed to copy to clipboard:", error);
        toast.error("Failed to copy to clipboard");
      });
  };

  return (
    <>
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
      <div
        className={cn(
          "rounded-lg border bg-white dark:bg-base-12 border-accent-4 text-grayA-11 w-[150px] px-2 py-1 flex gap-2 items-center cursor-pointer h-[28px] group-hover:border-grayA-3 font-mono",
          selected && "border-grayA-3",
        )}
        onClick={(e) => handleClick(e)}
      >
        <div>
          <CircleLock iconSize="sm-regular" className="text-gray-9" />
        </div>
        <div>{displayValue}</div>
      </div>
    </>
  );
};
