import { formatNumber } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import { Card, CardContent, CopyButton } from "@unkey/ui";
import { formatOutcomeName, getOutcomeColor } from "../../../../../utils";

export const OutcomeDistributionSection = ({
  outcomeCounts,
}: {
  outcomeCounts: Record<string, number>;
}) => {
  const outcomeEntries = Object.entries(outcomeCounts).filter(([_, count]) => count > 0);

  if (outcomeEntries.length === 0) {
    return null;
  }

  const getTextToCopy = () => {
    return outcomeEntries
      .map(([outcome, count]) => `${formatOutcomeName(outcome)}: ${count}`)
      .join("\n");
  };

  return (
    <div className="flex flex-col gap-1 mt-[16px]">
      <div className="flex justify-between items-center">
        <span className="text-[13px] text-accent-9 font-sans">
          Outcomes ({outcomeEntries.length})
        </span>
      </div>
      <Card className="bg-gray-2 border-gray-4 rounded-lg">
        <CardContent className="py-2 px-3 text-xs relative group">
          <div className="flex flex-col gap-1 whitespace-pre-wrap leading-relaxed">
            {outcomeEntries.map(([outcome, count]) => (
              <div className="group flex items-center w-full p-[3px]" key={outcome}>
                <div className="flex items-center text-left text-accent-9 whitespace-nowrap">
                  <div
                    className={cn(
                      "size-[10px] rounded-[2px] shadow-sm mr-2",
                      getOutcomeColor(outcome),
                    )}
                  />
                  <span>{formatOutcomeName(outcome)}:</span>
                </div>
                <span className="ml-2 text-xs text-accent-12 truncate font-mono tabular-nums">
                  {formatNumber(count)}
                </span>
              </div>
            ))}
          </div>
          <CopyButton
            value={getTextToCopy()}
            shape="square"
            variant="primary"
            size="2xlg"
            className="absolute bottom-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity rounded-md p-4"
            aria-label="Copy content"
          />
        </CardContent>
      </Card>
    </div>
  );
};
