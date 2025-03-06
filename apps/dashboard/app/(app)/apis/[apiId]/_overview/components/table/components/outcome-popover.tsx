"use client";

import { Badge } from "@/components/ui/badge";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { ChevronRight } from "@unkey/icons";
import { Button } from "@unkey/ui";

const formatOutcomeName = (outcome: string): string => {
  if (!outcome) {
    return "Unknown";
  }

  return outcome
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(" ");
};

const getOutcomeBadgeClass = (outcome: string): string => {
  switch (outcome) {
    case "VALID":
      return "bg-accent-4 text-accent-11";
    case "RATE_LIMITED":
      return "bg-warning-4 text-warning-11";
    case "INSUFFICIENT_PERMISSIONS":
    case "FORBIDDEN":
      return "bg-error-4 text-error-11";
    case "DISABLED":
    case "EXPIRED":
    case "USAGE_EXCEEDED":
      return "bg-gray-4 text-gray-11";
    default:
      return "bg-accent-4 text-accent-11";
  }
};

// Using compact formatter from your code
const compactFormatter = new Intl.NumberFormat("en-US", {
  notation: "compact",
  maximumFractionDigits: 1,
});

type OutcomesPopoverProps = {
  outcomeCounts: Record<string, number>;
  displayLimit?: number;
};

export const OutcomesPopover = ({ outcomeCounts, displayLimit = 2 }: OutcomesPopoverProps) => {
  const sortedOutcomeEntries = Object.entries(outcomeCounts)
    .filter(([_, count]) => count > 0)
    .sort(([outcome1], [outcome2]) => {
      if (outcome1 === "VALID") {
        return 1;
      }
      if (outcome2 === "VALID") {
        return -1;
      }
      return 0;
    });

  // Separate non-VALID outcomes and VALID outcome
  const nonValidOutcomes = sortedOutcomeEntries.filter(([outcome]) => outcome !== "VALID");
  const validOutcome = sortedOutcomeEntries.find(([outcome]) => outcome === "VALID");

  if (nonValidOutcomes.length <= displayLimit) {
    const displayOutcomes = validOutcome ? [...nonValidOutcomes, validOutcome] : nonValidOutcomes;

    return (
      <div className="flex flex-wrap gap-1">
        {displayOutcomes.map(([outcome, count]) => (
          <Badge
            key={outcome}
            className={cn(
              "px-[6px] rounded-md font-mono whitespace-nowrap",
              getOutcomeBadgeClass(outcome),
            )}
            title={`${count.toLocaleString()} ${formatOutcomeName(outcome)} requests`}
          >
            {formatOutcomeName(outcome)}: {compactFormatter.format(count)}
          </Badge>
        ))}
      </div>
    );
  }

  const displayedOutcomes = nonValidOutcomes.slice(0, displayLimit);
  const popoverOutcomes = nonValidOutcomes.slice(displayLimit);

  if (validOutcome) {
    popoverOutcomes.push(validOutcome);
  }

  const totalAdditionalOutcomes = popoverOutcomes.reduce((sum, [_, count]) => sum + count, 0);

  return (
    <div className="flex flex-wrap gap-1 items-center">
      {displayedOutcomes.map(([outcome, count]) => (
        <Badge
          key={outcome}
          className={cn(
            "px-[6px] rounded-md font-mono whitespace-nowrap",
            getOutcomeBadgeClass(outcome),
          )}
          title={`${count.toLocaleString()} ${formatOutcomeName(outcome)} requests`}
        >
          {formatOutcomeName(outcome)}: {compactFormatter.format(count)}
        </Badge>
      ))}

      {/* Popover for additional outcomes */}
      <Popover>
        <PopoverTrigger>
          <Button
            variant="ghost"
            size="sm"
            className="h-[22px] rounded-md px-2 text-xs font-medium text-accent-11 bg-gray-4 hover:bg-gray-5 flex items-center gap-1 cursor-pointer group relative font-mono [&_svg]:size-3"
            title="View all outcomes"
          >
            <span className="flex items-center">
              +{popoverOutcomes.length} more ({compactFormatter.format(totalAdditionalOutcomes)})
              <ChevronRight size="sm-regular" />
            </span>
          </Button>
        </PopoverTrigger>
        <PopoverContent
          className="min-w-60 bg-gray-1 dark:bg-black drop-shadow-2xl p-2 border-gray-6 rounded-lg"
          align="start"
          sideOffset={5}
        >
          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between border-b border-gray-4 pb-2">
              <div className="text-xs font-medium text-accent-12 px-1">Outcomes</div>
              <div className="text-xs text-gray-9">
                {Object.keys(outcomeCounts).filter((outcome) => outcomeCounts[outcome] > 0).length}{" "}
                types
              </div>
            </div>

            {/* Outcomes list */}
            <div className="flex flex-col gap-1.5">
              {popoverOutcomes.map(([outcome, count]) => (
                <div key={outcome} className="flex justify-between items-center px-1">
                  <Badge
                    className={cn(
                      "px-[6px] rounded-md font-mono whitespace-nowrap",
                      getOutcomeBadgeClass(outcome),
                    )}
                  >
                    {formatOutcomeName(outcome)}
                  </Badge>
                  <span className="text-xs font-mono text-accent-11">{count.toLocaleString()}</span>
                </div>
              ))}
            </div>
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
};
