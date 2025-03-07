"use client";
import { Badge } from "@/components/ui/badge";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { ChevronRight } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { STATUS_STYLES } from "../utils/get-row-class";

const formatOutcomeName = (outcome: string): string => {
  if (!outcome) {
    return "Unknown";
  }
  return outcome
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(" ");
};

const getOutcomeColor = (outcome: string): string => {
  switch (outcome) {
    case "VALID":
      return "bg-accent-9";
    case "RATE_LIMITED":
      return "bg-warning-9";
    case "INSUFFICIENT_PERMISSIONS":
    case "FORBIDDEN":
      return "bg-error-9";
    case "DISABLED":
      return "bg-gray-9";
    case "EXPIRED":
      return "bg-orange-9";
    case "USAGE_EXCEEDED":
      return "bg-violet-9";
    default:
      return "bg-accent-9";
  }
};

const getOutcomeBadgeStyle = (outcome: string): string => {
  switch (outcome) {
    case "RATE_LIMITED":
      return "bg-warning-4 text-warning-11 group-hover:bg-warning-5";
    case "INSUFFICIENT_PERMISSIONS":
    case "FORBIDDEN":
      return "bg-error-4 text-error-11 group-hover:bg-error-5";
    case "DISABLED":
      return "bg-gray-4 text-gray-11 group-hover:bg-gray-5";
    case "EXPIRED":
      return "bg-orange-4 text-orange-11 group-hover:bg-orange-5";
    case "USAGE_EXCEEDED":
      return "bg-violet-4 text-violet-11 group-hover:bg-violet-5";
    default:
      return "bg-gray-4 text-accent-11 hover:bg-gray-5 group-hover:text-accent-12";
  }
};

const compactFormatter = new Intl.NumberFormat("en-US", {
  notation: "compact",
  maximumFractionDigits: 1,
});

type OutcomesPopoverProps = {
  outcomeCounts: Record<string, number>;
  isSelected: boolean;
};

export const OutcomesPopover = ({ outcomeCounts, isSelected }: OutcomesPopoverProps) => {
  const allOutcomeEntries = Object.entries(outcomeCounts).filter(([_, count]) => count > 0);

  const nonValidOutcomes = allOutcomeEntries.filter(([outcome]) => outcome !== "VALID");

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
          <span className="tabular-nums flex-shrink-0 ml-1">{compactFormatter.format(count)}</span>
        </div>
      </Badge>
    );
  }

  return (
    <div className="flex flex-wrap gap-1 items-center">
      <Popover>
        <PopoverTrigger>
          <Button
            variant="ghost"
            size="sm"
            className={cn(
              containerStyle,
              "text-accent-11 bg-gray-4 hover:bg-gray-5 [&_svg]:size-3",
              isSelected
                ? STATUS_STYLES.success.badge.selected
                : STATUS_STYLES.success.badge.default,
            )}
            title="View all outcomes"
          >
            <div className="flex justify-between w-full items-center">
              <span className="overflow-hidden text-ellipsis whitespace-nowrap pr-1 max-w-[90px]">
                +{nonValidOutcomes.length} Outcomes
              </span>
              <ChevronRight size="sm-regular" className="flex-shrink-0" />
            </div>
          </Button>
        </PopoverTrigger>
        <PopoverContent
          className="min-w-64 bg-gray-1 dark:bg-black shadow-2xl p-0 border border-gray-6 rounded-lg overflow-hidden"
          align="start"
          sideOffset={5}
        >
          <div className="px-3 pt-3 ">
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
                      className={`size-[10px] ${getOutcomeColor(outcome)} rounded-[2px] shadow-sm`}
                    />
                    <span className="text-accent-12 text-xs font-medium">
                      {formatOutcomeName(outcome)}
                    </span>
                  </div>
                  <span className="text-accent-11 text-xs font-mono px-1.5 py-0.5 rounded tabular-nums">
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
};
