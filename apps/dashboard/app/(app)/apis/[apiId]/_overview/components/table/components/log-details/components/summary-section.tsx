"use client";
import { Card, CardContent } from "@/components/ui/card";
import { toast } from "@/components/ui/toaster";
import { Check, Clock, Clone, XMark } from "@unkey/icons";
import { Button } from "@unkey/ui";

export const SummarySection = ({
  summaryStats,
}: {
  summaryStats: string[];
}) => {
  const handleClick = () => {
    navigator.clipboard
      .writeText(summaryStats.join("\n"))
      .then(() => {
        toast.success("Summary copied to clipboard");
      })
      .catch((error) => {
        console.error("Failed to copy to clipboard:", error);
        toast.error("Failed to copy to clipboard");
      });
  };

  const getIconForStat = (stat: string) => {
    if (stat.includes("Valid")) {
      return <Check className="text-success-11 mr-2 flex-shrink-0" />;
    }
    if (stat.includes("Error")) {
      return <XMark className="text-error-10 mr-2 flex-shrink-0" />;
    }
    if (stat.includes("Age")) {
      return <Clock className="text-accent-10 mr-2 flex-shrink-0" />;
    }
    return null;
  };

  return (
    <div className="flex flex-col gap-1 mt-4">
      <div className="flex justify-between items-center">
        <span className="text-[13px] text-accent-9 font-sans">Summary</span>
      </div>
      <Card className="bg-gray-2 border-gray-4 rounded-lg">
        <CardContent className="py-2 px-3 text-xs relative group">
          <pre className="flex flex-col gap-1 whitespace-pre-wrap leading-relaxed">
            {summaryStats.map((stat, index) => {
              const [key, value] = stat.split(":");
              return (
                <div
                  className="group flex items-center w-full p-[3px]"
                  // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
                  key={index}
                >
                  {getIconForStat(stat)}
                  <span className="text-left text-accent-9 whitespace-nowrap">{key}:</span>
                  <span className="ml-2 text-xs text-accent-12 truncate">{value.trim()}</span>
                </div>
              );
            })}
          </pre>
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
