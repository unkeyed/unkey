import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { useVirtualizer } from "@tanstack/react-virtual";
import { ScrollText } from "lucide-react";
import { useRef, useState } from "react";
import type { Log } from "../types";
import { LogDetails } from "./log-details";

const TABLE_BORDER_THICKNESS = 1;
const ROW_HEIGHT = 26;

export const LogsTable = ({ logs }: { logs?: Log[] }) => {
  const [selectedLog, setSelectedLog] = useState<Log | null>(null);
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const parentRef = useRef<HTMLDivElement>(null);

  const virtualizer = useVirtualizer({
    count: logs?.length ?? 0,
    getScrollElement: () => parentRef.current,
    estimateSize: () => ROW_HEIGHT,
    overscan: 5,
  });

  const handleLogSelection = (log: Log) => {
    setSelectedLog(log);
    setTableDistanceToTop(
      document.getElementById("log-table")!.getBoundingClientRect().top +
        window.scrollY -
        TABLE_BORDER_THICKNESS,
    );
  };

  return (
    <div className="w-full">
      <div className="grid grid-cols-[166px_72px_20%_1fr] text-sm font-medium text-[#666666]">
        <div className="p-2 flex items-center">Time</div>
        <div className="p-2 flex items-center">Status</div>
        <div className="p-2 flex items-center">Request</div>
        <div className="p-2 flex items-center">Message</div>
      </div>
      <div className="w-full border-t border-border" />

      <div className="h-[75vh] overflow-auto" id="log-table" ref={parentRef}>
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
          <div
            style={{
              height: `${virtualizer.getTotalSize()}px`,
              width: "100%",
              position: "relative",
            }}
          >
            {virtualizer.getVirtualItems().map((virtualRow) => {
              const l = logs[virtualRow.index];
              return (
                <div
                  key={virtualRow.key}
                  data-index={virtualRow.index}
                  ref={virtualizer.measureElement}
                  onClick={() => handleLogSelection(l)}
                  role="button"
                  tabIndex={virtualRow.index}
                  aria-selected={selectedLog?.request_id === l.request_id}
                  onKeyDown={(event) => {
                    // Handle Enter or Space key press
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      handleLogSelection(l);
                    }

                    // Add arrow key navigation
                    if (event.key === "ArrowDown") {
                      event.preventDefault();
                      const nextElement = document.querySelector(
                        `[data-index="${virtualRow.index + 1}"]`,
                      ) as HTMLElement;
                      nextElement?.focus();
                    }
                    if (event.key === "ArrowUp") {
                      event.preventDefault();
                      const prevElement = document.querySelector(
                        `[data-index="${virtualRow.index - 1}"]`,
                      ) as HTMLElement;
                      prevElement?.focus();
                    }
                  }}
                  className={cn(
                    "font-mono grid grid-cols-[166px_72px_20%_1fr] text-[13px] leading-[14px] mb-[1px] rounded-[5px] h-[26px] cursor-pointer absolute top-0 left-0 w-full",
                    "hover:bg-background-subtle/90 pl-1",
                    {
                      "bg-amber-2 text-amber-11 hover:bg-amber-3":
                        l.response_status >= 400 && l.response_status < 500,
                      "bg-red-2 text-red-11 hover:bg-red-3": l.response_status >= 500,
                    },
                    selectedLog && {
                      "opacity-50": selectedLog.request_id !== l.request_id,
                      "opacity-100": selectedLog.request_id === l.request_id,
                      "bg-background-subtle/90":
                        selectedLog.request_id === l.request_id &&
                        l.response_status >= 200 &&
                        l.response_status < 300,
                      "bg-amber-3":
                        selectedLog.request_id === l.request_id &&
                        l.response_status >= 400 &&
                        l.response_status < 500,
                      "bg-red-3":
                        selectedLog.request_id === l.request_id && l.response_status >= 500,
                    },
                  )}
                  style={{
                    transform: `translateY(${virtualRow.start}px)`,
                  }}
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
                        "uppercase",
                      )}
                    >
                      {l.response_status}
                    </Badge>
                  </div>
                  <div className="px-[2px] flex items-center gap-2">
                    <Badge
                      className={cn(
                        "bg-background border border-solid border-border text-current hover:bg-transparent",
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
            })}
          </div>
        )}
        <LogDetails
          log={selectedLog}
          onClose={() => setSelectedLog(null)}
          distanceToTop={tableDistanceToTop}
        />
      </div>
    </div>
  );
};
