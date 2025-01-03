"use client";

import { TimestampInfo } from "@/components/timestamp-info";
import { type Column, VirtualTable } from "@/components/virtual-table";
import { cn } from "@/lib/utils";
import type { Log } from "@unkey/clickhouse/src/logs";
import { useState } from "react";
import { LogDetails } from "./log-details";
import { generateMockLogs } from "./utils";
import { Badge } from "@/components/ui/badge";
import { TriangleWarning2 } from "@unkey/icons";

const logs = generateMockLogs(10);
export const LogsTable = () => {
  const [selectedLog, setSelectedLog] = useState<Log | null>(null);

  const columns: Column<Log>[] = [
    {
      key: "time",
      header: "Time",
      width: "165px",
      headerClassName: "pl-9",
      render: (log) => (
        <div className="flex items-center gap-3 px-2">
          <TriangleWarning2
            className={cn(
              "size-3",
              log.response_status < 300 && "invisible",
              log.response_status >= 400 &&
                log.response_status < 500 &&
                "text-warning-11",
              log.response_status >= 500 && "text-error-11"
            )}
          />
          <TimestampInfo value={log.time} />
        </div>
      ),
    },
    {
      key: "status",
      header: "Status",
      width: "78px",
      render: (log) => (
        <Badge
          className={cn("uppercase px-[6px] rounded-md font-mono", {
            "bg-accent-4 text-accent-11 group-hover:bg-accent-5":
              log.response_status >= 200 && log.response_status < 300,
            "bg-warning-4 text-warning-11 group-hover:bg-warning-5":
              log.response_status >= 400 && log.response_status < 500,
            "bg-error-4 text-error-11 group-hover:bg-error-5":
              log.response_status >= 500,
          })}
        >
          {log.response_status}
        </Badge>
      ),
    },
    {
      key: "method",
      header: "Method",
      width: "78px",
      render: (log) => (
        <Badge className="uppercase px-[6px] rounded-md font-mono bg-accent-4 text-accent-11 group-hover:text-accent-12 group-hover:bg-accent-5">
          {log.method}
        </Badge>
      ),
    },
    {
      key: "path",
      header: "Path",
      width: "15%",
      render: (log) => (
        <div className="flex items-center gap-2 font-mono truncate">
          {log.path}
        </div>
      ),
    },
    {
      key: "response",
      header: "Response Body",
      width: "1fr",
      render: (log) => (
        <span className="truncate font-mono">{log.response_body}</span>
      ),
    },
  ];

  const getRowClassName = (log: Log) =>
    cn("text-accent-9", {
      "hover:text-accent-11":
        log.response_status >= 200 && log.response_status < 300,
      "bg-warning-2 text-warning-11 hover:bg-warning-3":
        log.response_status >= 400 && log.response_status < 500,
      "bg-error-2 text-error-11 hover:bg-error-3": log.response_status >= 500,
    });

  const getSelectedClassName = (log: Log, isSelected: boolean) =>
    isSelected
      ? cn({
          "text-accent-11 bg-accent-3":
            log.response_status >= 200 && log.response_status < 300,
          "bg-warning-3 text-warning-11":
            log.response_status >= 400 && log.response_status < 500,
          "bg-error-3 text-error-11": log.response_status >= 500,
        })
      : "";

  return (
    <VirtualTable
      data={logs ?? []}
      columns={columns}
      onRowClick={setSelectedLog}
      selectedItem={selectedLog}
      keyExtractor={(log) => log.request_id}
      rowClassName={getRowClassName}
      selectedClassName={getSelectedClassName}
      renderDetails={(log, onClose, distanceToTop) => (
        <LogDetails log={log} onClose={onClose} distanceToTop={distanceToTop} />
      )}
    />
  );
};
