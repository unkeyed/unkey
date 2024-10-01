"use client";

import { cn } from "@/lib/utils";
import { ScrollArea } from "@radix-ui/react-scroll-area";
import { format } from "date-fns";
import { useState } from "react";
import { RED_STATES, YELLOW_STATES } from "./constants";
import type { Log } from "./data";
import { LogsFilters } from "./filters";
import { LogDetails } from "./log-details";
import { getOutcomeIfValid } from "./utils";

type Props = {
  logs: Log[];
};

const TABLE_BORDER_THICKNESS = 1;

export default function LogsPage({ logs }: Props) {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);
  const [log, selectedLog] = useState<Log | null>(null);

  const handleLogSelection = (log: Log) => {
    selectedLog(log);
    // Since log details is fixed to the screen we have to calculate the distance from table to top of the screen.
    setTableDistanceToTop(
      document.getElementById("log-table")!.getBoundingClientRect().top +
        window.scrollY -
        TABLE_BORDER_THICKNESS
    );
  };

  return (
    <div className="flex flex-col gap-4 items-start w-full overflow-y-hidden">
      <LogsFilters />
      <div className="w-full">
        <div className="grid grid-cols-[166px_72px_20%_20%_1fr] text-sm font-medium text-[#666666]">
          <div className="p-2 flex items-center">Time</div>
          <div className="p-2 flex items-center">Status</div>
          <div className="p-2 flex items-center">Host</div>
          <div className="p-2 flex items-center">Request</div>
          <div className="p-2 flex items-center">Message</div>
        </div>
        <div className="w-full border-t border-border" />
        <ScrollArea className="h-[75vh]" id="log-table">
          {logs.map((log, index) => {
            const outcome = getOutcomeIfValid(log);
            return (
              // biome-ignore lint/a11y/useKeyWithClickEvents: don't know what to do atm
              <div
                key={`${log.request_id}#${index}`}
                onClick={() => handleLogSelection(log)}
                className={cn(
                  "font-mono grid grid-cols-[166px_72px_20%_20%_1fr] text-[13px] leading-[14px] mb-[1px] rounded-[5px] h-[26px] cursor-pointer ",
                  "hover:bg-background-subtle/50 data-[state=selected]:bg-background-subtle pl-1",
                  {
                    "bg-amber-2 text-amber-11  hover:bg-amber-3":
                      YELLOW_STATES.includes(outcome),
                    "bg-red-2 text-red-11  hover:bg-red-3":
                      RED_STATES.includes(outcome),
                  }
                )}
              >
                <div className="px-[2px] flex items-center">
                  {format(log.time, "MMM dd HH:mm:ss.SS")}
                </div>
                <div className="px-[2px] flex items-center">
                  {log.response_status}
                </div>
                <div className="px-[2px] flex items-center">{log.host}</div>
                <div className="px-[2px] flex items-center">{log.path}</div>
                <div className="px-[2px] flex items-center  w-[600px]">
                  <span className="truncate">{log.response_body}</span>
                </div>
              </div>
            );
          })}
          <LogDetails
            log={log}
            onClose={() => selectedLog(null)}
            distanceToTop={tableDistanceToTop}
          />
        </ScrollArea>
      </div>
    </div>
  );
}
