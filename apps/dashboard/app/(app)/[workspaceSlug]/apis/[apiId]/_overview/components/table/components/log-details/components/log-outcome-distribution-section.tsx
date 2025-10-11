import { formatNumber } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import { CopyButton } from "@unkey/ui";
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
    <div className="flex flex-col gap-1 mt-[16px] px-4">
      <div className="border bg-gray-2 border-gray-4 rounded-[10px] relative group">
        <div className="text-gray-11 text-[12px] leading-6 px-[14px] py-1.5 font-sans">
          Outcomes ({outcomeEntries.length})
        </div>
        <div className="border-gray-4 border-t rounded-[10px] bg-white dark:bg-black px-3.5 py-2">
          <div className="flex flex-col gap-1 whitespace-pre-wrap leading-relaxed">
            {outcomeEntries.map(([outcome, count]) => (
              <div className="flex items-center w-full px-[3px] h-6" key={outcome}>
                <div className="flex items-center text-left text-gray-11 whitespace-nowrap">
                  <div
                    className={cn(
                      "size-[10px] rounded-[2px] shadow-sm mr-2",
                      getOutcomeColor(outcome),
                    )}
                  />
                  <span className="text-xs text-gray-11 ">{formatOutcomeName(outcome)}:</span>
                </div>
                <span className="ml-2 text-xs text-accent-12 truncate font-mono tabular-nums">
                  {formatNumber(count)}
                </span>
              </div>
            ))}
          </div>
        </div>
        <CopyButton
          value={getTextToCopy()}
          shape="square"
          variant="outline"
          className="absolute bottom-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity rounded-md p-4 bg-gray-2 hover:bg-gray-2 size-2"
          aria-label="Copy content"
        />
      </div>
    </div>
  );
};
