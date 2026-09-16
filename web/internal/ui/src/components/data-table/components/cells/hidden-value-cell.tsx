import { IconCircleLockOutline18 } from "@unkey/icons";
// biome-ignore lint/correctness/noUnusedImports: React is needed for JSX
import React from "react";
import type { MouseEvent } from "react";
import { cn } from "../../../../lib/utils";
import { toast } from "../../../toaster";

const MASK_FILLER = "•".repeat(32);

export interface HiddenValueCellProps {
  prefix?: string;
  start: string;
  end?: string;
  title: string;
  selected: boolean;
}

export const HiddenValueCell = ({
  prefix = "",
  start,
  end = "",
  title = "Value",
  selected,
}: HiddenValueCellProps) => {
  const head = `${prefix ? `${prefix}_` : ""}${start}`;
  const displayValue = `${head}••••${end}`;

  const handleClick = (e: MouseEvent<HTMLDivElement>) => {
    e.stopPropagation();
    navigator.clipboard
      .writeText(displayValue)
      .then(() => {
        toast.success(`${title} copied to clipboard`);
      })
      .catch((error) => {
        console.error("Failed to copy to clipboard:", error);
        toast.error("Failed to copy to clipboard");
      });
  };

  return (
    // biome-ignore lint/a11y/useKeyWithClickEvents: copy is a pointer convenience; the row is the keyboard target
    <div
      className={cn(
        "rounded-lg border bg-white dark:bg-base-12 border-accent-4 text-grayA-11 w-[264px] whitespace-nowrap px-2 py-1 flex gap-2 items-center cursor-pointer h-[28px] group-hover:border-grayA-3 font-mono",
        selected && "border-grayA-3",
      )}
      onClick={handleClick}
    >
      <IconCircleLockOutline18 className="size-3 text-gray-9 shrink-0" />
      <span className="shrink-0">{head}</span>
      <span
        aria-hidden
        className="flex-1 min-w-[4ch] overflow-hidden text-grayA-8 tracking-[0.1em]"
      >
        {MASK_FILLER}
      </span>
      <span className="shrink-0">{end}</span>
    </div>
  );
};
