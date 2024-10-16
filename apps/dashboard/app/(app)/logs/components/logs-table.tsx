import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import { ScrollText } from "lucide-react";
import { useState } from "react";
import type { Log } from "../types";
import { LogDetails } from "./log-details";

const TABLE_BORDER_THICKNESS = 1;

export const LogsTable = ({ logs }: { logs?: Log[] }) => {
  const [selectedLog, setsSelectedLog] = useState<Log | null>(null);
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const handleLogSelection = (log: Log) => {
    setsSelectedLog(log);
    // Since log detail modal is fixed to the screen we have to calculate the distance from table to top of the screen.
    setTableDistanceToTop(
      document.getElementById("log-table")!.getBoundingClientRect().top +
        window.scrollY -
        TABLE_BORDER_THICKNESS
    );
  };

  return (
    <div className="w-full">
      <div className="grid grid-cols-[166px_72px_20%_1fr] text-sm font-medium text-[#666666]">
        <div className="p-2 flex items-center">Time</div>
        <div className="p-2 flex items-center">Status</div>
        {/* <div className="p-2 flex items-center">Host</div> */}
        <div className="p-2 flex items-center">Request</div>
        <div className="p-2 flex items-center">Message</div>
      </div>
      <div className="w-full border-t border-border" />
      {/* {logs.isLoading ? ( */}
      <ScrollArea className="h-[75vh] overflow-auto" id="log-table">
        {logs?.length === 0 || !logs ? (
          <div className="flex justify-center items-center h-[75vh]">
            <Card className="w-[400px] bg-background-subtle">
              <CardContent className="flex justify-center gap-2">
                <ScrollText />
                <div className="text-sm text-[#666666]">
                  There are no runtime logs in this time range
                </div>
              </CardContent>
            </Card>
          </div>
        ) : (
          logs.map((l, index) => {
            return (
              // biome-ignore lint/a11y/useKeyWithClickEvents: don't know what to do atm
              <div
                key={`${l.request_id}#${index}`}
                onClick={() => handleLogSelection(l)}
                className={cn(
                  "font-mono grid grid-cols-[166px_72px_20%_1fr] text-[13px] leading-[14px] mb-[1px] rounded-[5px] h-[26px] cursor-pointer ",
                  "hover:bg-background-subtle/90 pl-1",
                  // Conditional styling based on outcome
                  {
                    // Amber styling for yellow states
                    "bg-amber-2 text-amber-11 hover:bg-amber-3":
                      l.response_status >= 400 && l.response_status < 500,
                    // Red styling for red states
                    "bg-red-2 text-red-11 hover:bg-red-3":
                      l.response_status >= 500,
                  },

                  selectedLog && {
                    // Reduce opacity for non-selected logs
                    "opacity-50": selectedLog.request_id !== l.request_id,
                    // Full opacity for selected log
                    "opacity-100": selectedLog.request_id === l.request_id,
                    // Background for selected log (default state)
                    "bg-background-subtle/90":
                      selectedLog.request_id === l.request_id &&
                      l.response_status >= 200 &&
                      l.response_status < 300,
                    // Background for selected log (yellow state)
                    "bg-amber-3":
                      selectedLog.request_id === l.request_id &&
                      l.response_status >= 400 &&
                      l.response_status < 500,
                    // Background for selected log (red state)
                    "bg-red-3":
                      selectedLog.request_id === l.request_id &&
                      l.response_status >= 500,
                  }
                )}
              >
                <div className="px-[2px] flex items-center hover:underline hover:decoration-dotted">
                  <TimestampInfo value={l.time} />
                </div>
                <div className="px-[2px] flex items-center">
                  <Badge
                    className={cn(
                      {
                        "bg-background border border-solid border-border text-current hover:bg-transparent":
                          l.response_status >= 400,
                      },
                      "uppercase"
                    )}
                  >
                    {l.response_status}
                  </Badge>
                </div>
                <div className="px-[2px] flex items-center gap-1">
                  {" "}
                  <Badge
                    className={cn(
                      "bg-background border border-solid border-border text-current hover:bg-transparent"
                    )}
                  >
                    {l.method}
                  </Badge>
                  {l.path}
                </div>
                <div className="px-[2px] flex items-center max-w-[800px]">
                  <span className="truncate">{l.response_body}</span>
                </div>
              </div>
            );
          })
        )}
        <LogDetails
          log={selectedLog}
          onClose={() => setsSelectedLog(null)}
          distanceToTop={tableDistanceToTop}
        />
      </ScrollArea>
    </div>
  );
};
