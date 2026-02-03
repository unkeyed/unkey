"use client";

import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import { Badge, Empty, TimestampInfo } from "@unkey/ui";
import { useMemo } from "react";
import { useRuntimeLogs } from "../../context/runtime-logs-provider";
import { useRuntimeLogsFilters } from "../../hooks/use-runtime-logs-filters";
import type { RuntimeLog } from "../../types";
import { useRuntimeLogsQuery } from "./hooks/use-runtime-logs-query";
import { getRowClass } from "./utils/get-row-class";

export function RuntimeLogsTable() {
  const { filters } = useRuntimeLogsFilters();
  const { selectedLog, setSelectedLog } = useRuntimeLogs();
  const { logs, total, isLoading, hasMore, loadMore, isLoadingMore } = useRuntimeLogsQuery({
    filters,
  });

  const columns: Column<RuntimeLog>[] = useMemo(
    () => [
      {
        key: "time",
        header: "Time",
        width: "15%",
        render: (log) => (
          <TimestampInfo
            value={log.time}
            className="font-mono group-hover:underline decoration-dotted"
          />
        ),
      },
      {
        key: "severity",
        header: "Severity",
        width: "10%",
        render: (log) => {
          const variant =
            log.severity === "ERROR" ? "error" : log.severity === "WARN" ? "warning" : "success";
          return (
            <Badge variant={variant} className="uppercase px-2 rounded-md font-mono">
              {log.severity}
            </Badge>
          );
        },
      },
      {
        key: "message",
        header: "Message",
        width: "30%",
        render: (log) => (
          <span className="truncate block font-mono text-xs max-w-[350px]">{log.message}</span>
        ),
      },
      {
        key: "region",
        header: "Region",
        width: "15%",
        render: (log) => (
          <div className="font-mono pr-4 truncate uppercase" title={log.region}>
            {log.region}
          </div>
        ),
      },
    ],
    [],
  );

  const getRowClassName = (log: RuntimeLog) => {
    const isSelected = selectedLog?.time === log.time;
    const baseClass = getRowClass(log.severity);
    return cn(
      baseClass,
      isSelected && "bg-accent-3 text-accent-12",
      selectedLog && !isSelected && "opacity-50",
    );
  };

  if (isLoading) {
    return <div className="p-8 text-center text-grayA-11">Loading logs...</div>;
  }

  return (
    <VirtualTable
      data={logs}
      isLoading={isLoading}
      isFetchingNextPage={isLoadingMore}
      onLoadMore={loadMore}
      columns={columns}
      onRowClick={setSelectedLog}
      selectedItem={selectedLog}
      keyExtractor={(log) => `${log.time}-${log.k8s_pod_name}`}
      rowClassName={getRowClassName}
      loadMoreFooterProps={{
        hide: isLoading,
        buttonText: "Load more logs",
        hasMore,
        countInfoText: (
          <div className="flex gap-2">
            <span>Showing</span>
            <span className="text-accent-12">{new Intl.NumberFormat().format(logs.length)}</span>
            <span>of</span>
            <span className="text-accent-12">{new Intl.NumberFormat().format(total)}</span>
            <span>logs</span>
          </div>
        ),
      }}
      emptyState={
        <div className="w-full flex justify-center items-center h-full">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>No runtime logs</Empty.Title>
            <Empty.Description className="text-left">
              No logs found for the selected filters and time range.
            </Empty.Description>
          </Empty>
        </div>
      }
    />
  );
}
