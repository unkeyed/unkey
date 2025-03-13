import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { Clone } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { toast } from "sonner";
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

  const handleClick = () => {
    const formattedContent = outcomeEntries
      .map(([outcome, count]) => `${formatOutcomeName(outcome)}: ${count}`)
      .join("\n");

    navigator.clipboard
      .writeText(formattedContent)
      .then(() => {
        toast.success("Outcomes  copied to clipboard");
      })
      .catch((error) => {
        console.error("Failed to copy to clipboard:", error);
        toast.error("Failed to copy to clipboard");
      });
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
                  {count.toLocaleString()}
                </span>
              </div>
            ))}
          </div>
          <Button
            shape="square"
            onClick={handleClick}
            variant="outline"
            className="absolute bottom-2 right-3 opacity-0 group-hover:opacity-100 transition-opacity rounded-sm"
            aria-label="Copy content"
          >
            <Clone />
          </Button>
        </CardContent>
      </Card>
    </div>
  );
};
