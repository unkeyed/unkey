import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { useVirtualizer } from "@tanstack/react-virtual";
import { ScrollText } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useInterval } from "usehooks-ts";
import { useLogSearchParams } from "../query-state";
import type { Log } from "../types";
import { getTimeseriesGranularity } from "../utils";
import { LogDetails } from "./log-details";
import { LoadingRow } from "./logs-table-loading-row";

const TABLE_BORDER_THICKNESS = 1;
const ROW_HEIGHT = 26;
const SKELETON_ROWS = 50;

const roundToSecond = (timestamp: number) => Math.floor(timestamp / 1000) * 1000;

const useFetchLogs = (initialLogs: Log[]) => {
  const { searchParams } = useLogSearchParams();
  const [logs, setLogs] = useState(initialLogs);
  const [endTime, setEndTime] = useState(searchParams.endTime);

  useInterval(() => setEndTime(roundToSecond(Date.now())), searchParams.endTime ? null : 3000);

  const filters = useMemo(
    () => ({
      host: searchParams.host,
      requestId: searchParams.requestId,
      path: searchParams.path,
      method: searchParams.method,
      responseStatus: searchParams.responseStatus,
    }),
    [
      searchParams.host,
      searchParams.requestId,
      searchParams.path,
      searchParams.method,
      searchParams.responseStatus,
    ],
  );

  const hasFilters = useMemo(
    () =>
      Boolean(
        filters.host ||
          filters.requestId ||
          filters.path ||
          filters.method ||
          filters.responseStatus.length,
      ),
    [filters],
  );

  useInterval(() => setEndTime(Date.now()), searchParams.endTime ? null : 3000);

  const { startTime: rawStartTime, endTime: rawEndTime } = getTimeseriesGranularity(
    searchParams.startTime,
    endTime,
  );

  const startTime = roundToSecond(rawStartTime);
  const todoEndTime = roundToSecond(rawEndTime);

  const { data: newData, isLoading } = trpc.logs.queryLogs.useQuery(
    {
      limit: 100,
      startTime,
      endTime: todoEndTime,
      ...filters,
    },
    {
      refetchInterval: searchParams.endTime ? false : 3000,
      keepPreviousData: true,
    },
  );

  const updateLogs = useCallback(() => {
    if (hasFilters) {
      setLogs(newData ?? []);
      return;
    }

    if (!newData?.length) {
      return;
    }

    setLogs((prevLogs) => {
      const existingIds = new Set(prevLogs.map((log) => log.request_id));
      const uniqueNewLogs = newData.filter((newLog) => !existingIds.has(newLog.request_id));
      return [...uniqueNewLogs, ...prevLogs];
    });
  }, [newData, hasFilters]); // Reduced dependencies

  useEffect(() => {
    updateLogs();
  }, [updateLogs]);

  return { logs, isLoading };
};

export const LogsTable = ({ initialLogs }: { initialLogs?: Log[] }) => {
  const { logs, isLoading } = useFetchLogs(initialLogs ?? []);
  const [selectedLog, setSelectedLog] = useState<Log | null>(null);
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);
  const parentRef = useRef<HTMLDivElement>(null);
  const tableRef = useRef<HTMLDivElement>(null);

  const virtualizer = useVirtualizer({
    count: isLoading ? SKELETON_ROWS : logs?.length ?? 0,
    getScrollElement: () => parentRef.current,
    estimateSize: () => ROW_HEIGHT,
    overscan: 5,
  });

  const handleLogSelection = (log: Log) => {
    setSelectedLog(log);
    setTableDistanceToTop(
      tableRef.current?.getBoundingClientRect().top ?? 0 + window.scrollY - TABLE_BORDER_THICKNESS,
    );
  };

  return (
    <div className="w-full">
      <div className="grid grid-cols-[166px_72px_20%_1fr] text-sm font-medium text-[#666666]">
        <div className="p-2 flex items-center">Time</div>
        <div className="p-2 flex items-center">Status</div>
        <div className="p-2 flex items-center">Path</div>
        <div className="p-2 flex items-center">Response Body</div>
      </div>
      <div className="w-full border-t border-border" />

      <div
        className="h-[75vh] overflow-auto"
        ref={(el) => {
          //@ts-expect-error ts complaining for no reason
          parentRef.current = el;

          //@ts-expect-error ts complaining for no reason
          tableRef.current = el;
        }}
      >
        {!isLoading && (logs?.length === 0 || !logs) ? (
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
            className={cn("transition-opacity duration-300", {
              "opacity-40": isLoading,
            })}
            style={{
              height: `${virtualizer.getTotalSize()}px`,
              width: "100%",
              position: "relative",
            }}
          >
            {virtualizer.getVirtualItems().map((virtualRow) => {
              if (isLoading) {
                return (
                  <div
                    key={virtualRow.key}
                    style={{
                      position: "absolute",
                      top: `${virtualRow.start}px`,
                      width: "100%",
                    }}
                  >
                    <LoadingRow />
                  </div>
                );
              }

              const l = logs ? logs[virtualRow.index] : null;
              return (
                l && (
                  <div
                    key={virtualRow.key}
                    data-index={virtualRow.index}
                    ref={virtualizer.measureElement}
                    onClick={() => handleLogSelection(l)}
                    tabIndex={virtualRow.index}
                    aria-selected={selectedLog?.request_id === l.request_id}
                    onKeyDown={(event) => {
                      if (event.key === "Escape") {
                        setSelectedLog(null);
                      }
                      if (event.key === "Enter" || event.key === " ") {
                        event.preventDefault();
                        handleLogSelection(l);
                      }
                      if (event.key === "ArrowDown") {
                        //Without preventDefault table moves up and down as you navigate with keyboard
                        event.preventDefault();
                        const nextElement = document.querySelector(
                          `[data-index="${virtualRow.index + 1}"]`,
                        ) as HTMLElement;
                        nextElement?.focus();
                      }
                      if (event.key === "ArrowUp") {
                        //Without preventDefault table moves up and down as you navigate with keyboard
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
                      top: `${virtualRow.start}px`,
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
                )
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
