"use client";

import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import type { RatelimitLog } from "@unkey/clickhouse/src/ratelimits";
import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useMemo } from "react";
import { DEFAULT_REJECTED_FLAG } from "../../constants";
import { useRatelimitLogsContext } from "../../context/logs";
import { useRatelimitLogsQuery } from "./hooks/use-logs-query";
import { LogsTableAction } from "./logs-actions";

type StatusStyle = {
  base: string;
  hover: string;
  selected: string;
  badge: {
    default: string;
    selected: string;
  };
  focusRing: string;
};

const STATUS_STYLES = {
  success: {
    base: "text-accent-9",
    hover: "hover:text-accent-11 dark:hover:text-accent-12 hover:bg-accent-3",
    selected: "text-accent-11 bg-accent-3 dark:text-accent-12",
    badge: {
      default: "bg-accent-4 text-accent-11 group-hover:bg-accent-5",
      selected: "bg-accent-5 text-accent-12 hover:bg-hover-5",
    },
    focusRing: "focus:ring-accent-7",
  },
  error: {
    base: "text-error-11 bg-error-2",
    hover: "hover:bg-error-3",
    selected: "bg-error-3",
    badge: {
      default: "bg-error-4 text-error-11 group-hover:bg-error-5",
      selected: "bg-error-5 text-error-11 hover:bg-error-5",
    },
    focusRing: "focus:ring-error-7",
  },
};

const getStatusStyle = (rejected: number): StatusStyle => {
  if (rejected === DEFAULT_REJECTED_FLAG) {
    return STATUS_STYLES.success;
  }
  return STATUS_STYLES.error;
};

const getSelectedClassName = (log: RatelimitLog, isSelected: boolean) => {
  if (!isSelected) {
    return "";
  }
  const style = getStatusStyle(log.rejected);
  return style.selected;
};

export const RatelimitLogsTable = ({
  namespaceId,
}: {
  namespaceId: string;
}) => {
  const { setSelectedLog, selectedLog, isLive } = useRatelimitLogsContext();
  const { realtimeLogs, historicalLogs, isLoading, isLoadingMore, loadMore } =
    useRatelimitLogsQuery({
      namespaceId,
      startPolling: isLive,
      pollIntervalMs: 2000,
    });

  const getRowClassName = (log: RatelimitLog) => {
    const style = getStatusStyle(log.rejected);
    const isSelected = selectedLog?.request_id === log.request_id;

    return cn(
      style.base,
      style.hover,
      "group rounded-md",
      "focus:outline-none focus:ring-1 focus:ring-opacity-40",
      style.focusRing,
      isSelected && style.selected,
      isLive &&
        !realtimeLogs.some((realtime) => realtime.request_id === log.request_id) && [
          "opacity-50",
          "hover:opacity-100",
        ],
      selectedLog && {
        "opacity-50 z-0": !isSelected,
        "opacity-100 z-10": isSelected,
      },
    );
  };
  // biome-ignore lint/correctness/useExhaustiveDependencies: it's okay
  const columns: Column<RatelimitLog>[] = useMemo(
    () => [
      {
        key: "time",
        header: "Time",
        headerClassName: "pl-3",
        width: "10%",
        render: (log) => (
          <div className="flex items-center gap-3 pl-2 min-w-0 truncate">
            <TimestampInfo
              value={log.time}
              className={cn(
                "font-mono group-hover:underline decoration-dotted",
                selectedLog && selectedLog.request_id !== log.request_id && "pointer-events-none",
              )}
            />
          </div>
        ),
      },
      {
        key: "identifier",
        header: "Identifier",
        width: "15%",
        headerClassName: "pl-1",
        render: (log) => <div className="font-mono truncate mr-1">{log.identifier}</div>,
      },
      {
        key: "rejected",
        header: "Status",
        width: "10%",
        headerClassName: "pl-1",
        render: (log) => {
          const style = getStatusStyle(log.rejected);
          const isSelected = selectedLog?.request_id === log.request_id;
          return (
            <Badge
              className={cn(
                "uppercase px-[6px] rounded-md font-mono",
                isSelected ? style.badge.selected : style.badge.default,
              )}
            >
              {log.rejected ? "Rejected" : "Succeeded"}
            </Badge>
          );
        },
      },
      {
        key: "region",
        header: "Region",
        width: "auto",
        render: (log) => <div className="font-mono">{log.colo}</div>,
      },
      {
        key: "region",
        header: "",
        width: "12.5%",
        render: () => <LogsTableAction />,
      },
    ],
    [],
  );

  return (
    <VirtualTable
      data={historicalLogs}
      realtimeData={realtimeLogs}
      isLoading={isLoading}
      isFetchingNextPage={isLoadingMore}
      onLoadMore={loadMore}
      columns={columns}
      onRowClick={setSelectedLog}
      selectedItem={selectedLog}
      keyExtractor={(log) => log.request_id}
      rowClassName={getRowClassName}
      selectedClassName={getSelectedClassName}
      emptyState={
        <div className="w-full flex justify-center items-center h-full">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>Logs</Empty.Title>
            <Empty.Description className="text-left">
              Keep track of all activity within your workspace. Logs automatically record key
              actions like key creation, permission updates, and configuration changes, giving you a
              clear history of resource requests.
            </Empty.Description>
            <Empty.Actions className="mt-4 justify-start">
              <a
                href="https://www.unkey.com/docs/introduction"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button>
                  <BookBookmark />
                  Documentation
                </Button>
              </a>
            </Empty.Actions>
          </Empty>
        </div>
      }
    />
  );
};
