"use client";
import { Ban } from "@unkey/icons";
import type * as React from "react";
import { cn } from "../../../../lib/utils";
import { Badge } from "../../../badge";
import { OutcomePopoverCell, formatCompactNumber } from "./outcome-popover-cell";

export interface InvalidCountCellProps {
  count: number;
  outcomeCounts: Record<string, number>;
  isSelected: boolean;
  badgeClassName: string;
  title?: string;
  className?: string;
}

export function InvalidCountCell({
  count,
  outcomeCounts,
  isSelected,
  badgeClassName,
  title,
  className,
}: InvalidCountCellProps): React.ReactNode {
  return (
    <div className={cn("flex items-center w-full", className)}>
      <div className="shrink-0">
        <Badge
          className={cn(
            "px-1.5 rounded-md font-mono whitespace-nowrap flex items-center text-right tabular-nums",
            badgeClassName,
          )}
          title={title}
        >
          <span className="mr-1.5 shrink-0">
            <Ban iconSize="sm-regular" />
          </span>
          <span className="overflow-hidden text-ellipsis whitespace-nowrap w-11.25">
            {formatCompactNumber(count)}
          </span>
        </Badge>
      </div>
      <div className="ml-2 shrink-0">
        <OutcomePopoverCell outcomeCounts={outcomeCounts} isSelected={isSelected} />
      </div>
    </div>
  );
}
