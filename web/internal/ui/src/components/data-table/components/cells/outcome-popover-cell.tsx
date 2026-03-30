"use client";
import { ChevronRight } from "@unkey/icons";
import type * as React from "react";
import { cn } from "../../../../lib/utils";
import { Badge } from "../../../badge";
import { Button } from "../../../buttons/button";
import { Popover, PopoverContent, PopoverTrigger } from "../../../dialog/popover";

export function formatOutcomeName(outcome: string): string {
  if (!outcome) {
    return "Unknown";
  }
  return outcome
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(" ");
}

export function formatCompactNumber(n: number): string {
  return Intl.NumberFormat("en", { notation: "compact" }).format(n);
}

export const OUTCOME_BACKGROUND_COLORS: Record<string, string> = {
  VALID: "bg-success-9",
  RATE_LIMITED: "bg-warning-9",
  INSUFFICIENT_PERMISSIONS: "bg-error-9",
  FORBIDDEN: "bg-error-9",
  DISABLED: "bg-gray-9",
  EXPIRED: "bg-orange-9",
  USAGE_EXCEEDED: "bg-feature-9",
  UNKNOWN: "bg-accent-9",
};

export const OUTCOME_BADGE_STYLES: Record<string, string> = {
  VALID: "bg-gray-4 text-accent-11 hover:bg-gray-5 group-hover:text-accent-12",
  RATE_LIMITED: "bg-warning-4 text-warning-11 group-hover:bg-warning-5",
  INSUFFICIENT_PERMISSIONS: "bg-error-4 text-error-11 group-hover:bg-error-5",
  FORBIDDEN: "bg-error-4 text-error-11 group-hover:bg-error-5",
  DISABLED: "bg-gray-4 text-gray-11 group-hover:bg-gray-5",
  EXPIRED: "bg-orange-4 text-orange-11 group-hover:bg-orange-5",
  USAGE_EXCEEDED: "bg-feature-4 text-feature-11 group-hover:bg-feature-5",
  UNKNOWN: "bg-gray-4 text-accent-11 hover:bg-gray-5 group-hover:text-accent-12",
};

export function getOutcomeColor(outcome: string): string {
  return OUTCOME_BACKGROUND_COLORS[outcome] ?? OUTCOME_BACKGROUND_COLORS.UNKNOWN;
}

export function getOutcomeBadgeStyle(outcome: string): string {
  return OUTCOME_BADGE_STYLES[outcome] ?? OUTCOME_BADGE_STYLES.UNKNOWN;
}

const SELECTED_BADGE = "bg-gray-5 text-grayA-12 hover:bg-gray-5";
const DEFAULT_BADGE = "bg-gray-3 text-grayA-11 group-hover:bg-gray-5";

export interface OutcomePopoverCellProps {
  outcomeCounts: Record<string, number>;
  isSelected: boolean;
}

export function OutcomePopoverCell({
  outcomeCounts,
  isSelected,
}: OutcomePopoverCellProps): React.ReactNode {
  const nonValidOutcomes = Object.entries(outcomeCounts).filter(
    ([outcome, count]) => outcome !== "VALID" && count > 0,
  );

  if (nonValidOutcomes.length === 0) {
    return null;
  }

  const containerStyle = "h-[22px] rounded-md px-2 text-xs font-medium w-[110px] flex items-center";

  if (nonValidOutcomes.length === 1) {
    const [outcome, count] = nonValidOutcomes[0];
    return (
      <Badge
        className={cn(containerStyle, getOutcomeBadgeStyle(outcome))}
        title={`${count.toLocaleString()} ${formatOutcomeName(outcome)} requests`}
      >
        <div className="flex justify-between w-full items-center">
          <span className="overflow-hidden text-ellipsis whitespace-nowrap">
            {formatOutcomeName(outcome)}:
          </span>
          <span className="tabular-nums shrink-0 ml-1">{formatCompactNumber(count)}</span>
        </div>
      </Badge>
    );
  }

  return (
    <div className="flex flex-wrap gap-1 items-center">
      <Popover>
        <PopoverTrigger asChild onClick={(e) => e.stopPropagation()}>
          <Button
            variant="ghost"
            size="sm"
            className={cn(
              containerStyle,
              "text-accent-11 bg-gray-4 hover:bg-gray-5 [&_svg]:size-3",
              isSelected ? SELECTED_BADGE : DEFAULT_BADGE,
            )}
            title="View all outcomes"
          >
            <div className="flex justify-between w-full items-center">
              <span className="overflow-hidden text-ellipsis whitespace-nowrap pr-1 max-w-[90px]">
                +{nonValidOutcomes.length} Outcomes
              </span>
              <ChevronRight iconSize="sm-regular" className="shrink-0" />
            </div>
          </Button>
        </PopoverTrigger>
        <PopoverContent
          className="min-w-64 bg-gray-1 dark:bg-black shadow-2xl p-0 border border-gray-6 rounded-lg overflow-hidden"
          align="start"
          sideOffset={5}
        >
          <div className="px-3 pt-3">
            <div className="flex items-center justify-between">
              <div className="text-xs font-medium text-gray-9">Outcomes</div>
              <div className="text-xs text-gray-9">
                {nonValidOutcomes.length} {nonValidOutcomes.length === 1 ? "type" : "types"}
              </div>
            </div>
          </div>
          <div className="p-2">
            <div className="flex flex-col">
              {nonValidOutcomes.map(([outcome, count], index) => (
                <div
                  key={outcome}
                  className={cn("flex items-center justify-between py-1.5", index === 0 && "pt-1")}
                >
                  <div className="flex items-center gap-2.5 pl-1.5 font-mono">
                    <div
                      className={cn(
                        "size-[10px] rounded-[2px] shadow-xs",
                        getOutcomeColor(outcome),
                      )}
                    />
                    <span className="text-accent-12 text-xs font-medium">
                      {formatOutcomeName(outcome)}
                    </span>
                  </div>
                  <span className="text-accent-11 text-xs font-mono px-1.5 py-0.5 rounded-sm tabular-nums">
                    {count.toLocaleString()}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
