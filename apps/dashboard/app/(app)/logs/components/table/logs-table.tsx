import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { Log } from "@unkey/clickhouse/src/logs";
import { useState } from "react";
import { useFetchLogs } from "./hooks";
import { LogDetails } from "./log-details";
import { VirtualTable } from "@/components/virtual-table";
import { Column } from "@/components/virtual-table/types";

export const LogsTable = ({ initialLogs }: { initialLogs?: Log[] }) => {
  const { logs, isLoading, switchToLive } = useFetchLogs(initialLogs ?? []);
  const [selectedLog, setSelectedLog] = useState<Log | null>(null);

  const columns: Column<Log>[] = [
    {
      key: "time",
      header: "Time",
      width: "166px",
      render: (log) => <TimestampInfo value={log.time} />,
    },
    {
      key: "status",
      header: "Status",
      width: "72px",
      render: (log) => (
        <Badge
          className={cn(
            {
              "bg-background border border-solid border-border text-current hover:bg-transparent":
                log.response_status >= 400,
            },
            "uppercase"
          )}
        >
          {log.response_status}
        </Badge>
      ),
    },
    {
      key: "path",
      header: "Path",
      width: "20%",
      render: (log) => (
        <div className="flex items-center gap-2">
          <Badge className="bg-background border border-solid border-border text-current hover:bg-transparent">
            {log.method}
          </Badge>
          {log.path}
        </div>
      ),
    },
    {
      key: "response",
      header: "Response Body",
      width: "1fr",
      render: (log) => <span className="truncate">{log.response_body}</span>,
    },
  ];

  const getRowClassName = (log: Log) =>
    cn({
      "bg-amber-2 text-amber-11 hover:bg-amber-3":
        log.response_status >= 400 && log.response_status < 500,
      "bg-red-2 text-red-11 hover:bg-red-3": log.response_status >= 500,
    });

  const getSelectedClassName = (log: Log, isSelected: boolean) =>
    isSelected
      ? cn({
          "bg-background-subtle/90":
            log.response_status >= 200 && log.response_status < 300,
          "bg-amber-3": log.response_status >= 400 && log.response_status < 500,
          "bg-red-3": log.response_status >= 500,
        })
      : "";

  return (
    <VirtualTable
      data={logs ?? []}
      columns={columns}
      isLoading={isLoading}
      onLoadMore={switchToLive}
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
