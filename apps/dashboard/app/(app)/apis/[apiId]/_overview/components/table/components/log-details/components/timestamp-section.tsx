"use client";
import { TimestampInfo } from "@/components/timestamp-info";
import { Card, CardContent } from "@/components/ui/card";
import { toast } from "@/components/ui/toaster";
import { Button } from "@unkey/ui";
import { Copy } from "lucide-react";
// TimestampSection - displays a list of timestamps in a format similar to OutcomeDistributionSection
export const TimestampSection = ({
  timestamps,
  title,
}: {
  timestamps: Array<{
    label: string;
    value: string | number | null;
  }>;
  title: string;
}) => {
  // Filter out null or undefined timestamps
  const validTimestamps = timestamps.filter(
    (ts) => ts.value !== null && ts.value !== undefined
  );

  if (validTimestamps.length === 0) {
    return null;
  }

  const handleClick = () => {
    const formattedContent = validTimestamps
      .map((ts) => `${ts.label}: ${String(ts.value)}`)
      .join("\n");

    navigator.clipboard
      .writeText(formattedContent)
      .then(() => {
        toast.success(`${title} copied to clipboard`);
      })
      .catch((error) => {
        console.error("Failed to copy to clipboard:", error);
        toast.error("Failed to copy to clipboard");
      });
  };

  return (
    <div className="flex flex-col gap-1 mt-[16px]">
      <div className="flex justify-between items-center">
        <span className="text-[13px] text-accent-9 font-sans">{title}</span>
      </div>
      <Card className="bg-gray-2 border-gray-4 rounded-lg">
        <CardContent className="py-2 px-3 text-xs relative group">
          <div className="flex flex-col gap-1 whitespace-pre-wrap leading-relaxed">
            {validTimestamps.map((ts, index) => (
              <div
                className="group flex items-center w-full p-[3px]"
                key={`${ts.label}-${index}`}
              >
                <div className="flex items-center text-left text-accent-9 whitespace-nowrap">
                  <span>{ts.label}:</span>
                </div>
                <span className="ml-2 text-xs text-accent-12 truncate">
                  {ts.value ? (
                    typeof ts.value === "string" && !ts.value.match(/^\d/) ? (
                      ts.value
                    ) : (
                      <TimestampInfo value={ts.value} />
                    )
                  ) : (
                    "N/A"
                  )}
                </span>
              </div>
            ))}
          </div>
          <Button
            shape="square"
            onClick={handleClick}
            className="absolute bottom-2 right-3 opacity-0 group-hover:opacity-100 transition-opacity"
            aria-label={`Copy ${title.toLowerCase()}`}
          >
            <Copy size={14} />
          </Button>
        </CardContent>
      </Card>
    </div>
  );
};
