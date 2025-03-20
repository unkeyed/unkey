"use client";

import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import type { Log } from "@unkey/clickhouse/src/logs";
import { BookBookmark, TriangleWarning2 } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useMemo } from "react";
import { isDisplayProperty, useLogsContext } from "../../context/logs";
import { extractResponseField } from "../../utils";
import { useLogsQuery } from "./hooks/use-logs-query";

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
    base: "text-grayA-9",
    hover: "hover:text-accent-11 dark:hover:text-accent-12 hover:bg-grayA-3",
    selected: "text-accent-12 bg-grayA-3 hover:text-accent-12",
    badge: {
      default: "bg-grayA-3 text-grayA-11 group-hover:bg-grayA-5",
      selected: "bg-grayA-5 text-grayA-12 hover:bg-grayA-5",
    },
    focusRing: "focus:ring-accent-7",
  },
  warning: {
    base: "text-warning-11 bg-warning-2",
    hover: "hover:bg-warning-3",
    selected: "bg-warning-3",
    badge: {
      default: "bg-warning-4 text-warning-11 group-hover:bg-warning-5",
      selected: "bg-warning-5 text-warning-11 hover:bg-warning-5",
    },
    focusRing: "focus:ring-warning-7",
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

const getStatusStyle = (status: number): StatusStyle => {
  if (status >= 500) {
    return STATUS_STYLES.error;
  }
  if (status >= 400) {
    return STATUS_STYLES.warning;
  }
  return STATUS_STYLES.success;
};

const WARNING_ICON_STYLES = {
  base: "size-3",
  warning: "text-warning-11",
  error: "text-error-11",
};

const getSelectedClassName = (log: Log, isSelected: boolean) => {
  if (!isSelected) {
    return "";
  }
  const style = getStatusStyle(log.response_status);
  return style.selected;
};

const WarningIcon = ({ status }: { status: number }) => (
  <TriangleWarning2
    className={cn(
      WARNING_ICON_STYLES.base,
      status < 300 && "invisible",
      status >= 400 && status < 500 && WARNING_ICON_STYLES.warning,
      status >= 500 && WARNING_ICON_STYLES.error,
    )}
  />
);

const additionalColumns: Column<Log>[] = [
  "response_body",
  "request_body",
  "request_headers",
  "response_headers",
  "request_id",
  "workspace_id",
  "host",
].map((key) => ({
  key,
  header: key
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" "),
  width: "1fr",
  render: (log: Log) => (
    <div className="font-mono whitespace-nowrap truncate max-w-[500px]">
      {log[key as keyof Log]}
    </div>
  ),
}));

export const LogsTable = () => {
  const { displayProperties, setSelectedLog, selectedLog, isLive } = useLogsContext();
  const { realtimeLogs, historicalLogs, isLoading, isLoadingMore, loadMore } = useLogsQuery({
    startPolling: isLive,
    pollIntervalMs: 2000,
  });

  const getRowClassName = (log: Log) => {
    const style = getStatusStyle(log.response_status);
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
  const basicColumns: Column<Log>[] = useMemo(
    () => [
      {
        key: "time",
        header: "Time",
        width: "5%",
        render: (log) => (
          <TimestampInfo
            value={log.time}
            className={cn(
              "font-mono group-hover:underline decoration-dotted",
              selectedLog && selectedLog.request_id !== log.request_id && "pointer-events-none",
            )}
          />
        ),
      },
      {
        key: "response_status",
        header: "Status",
        width: "7.5%",
        render: (log) => {
          const style = getStatusStyle(log.response_status);
          const isSelected = selectedLog?.request_id === log.request_id;
          return (
            <Badge
              className={cn(
                "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
                isSelected ? style.badge.selected : style.badge.default,
              )}
            >
              {log.response_status}{" "}
              {extractResponseField(log, "code") ? `| ${extractResponseField(log, "code")}` : ""}
            </Badge>
          );
        },
      },
      {
        key: "method",
        header: "Method",
        width: "7.5%",
        render: (log) => {
          const isSelected = selectedLog?.request_id === log.request_id;
          return (
            <Badge
              className={cn(
                "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
                isSelected
                  ? STATUS_STYLES.success.badge.selected
                  : STATUS_STYLES.success.badge.default,
              )}
            >
              {log.method}
            </Badge>
          );
        },
      },
      {
        key: "path",
        header: "Path",
        width: "15%",
        render: (log) => <div className="font-mono pr-4">{log.path}</div>,
      },
    ],
    [selectedLog?.request_id],
  );

  const visibleColumns = useMemo(() => {
    const filtered = [...basicColumns, ...additionalColumns].filter(
      (column) => isDisplayProperty(column.key) && displayProperties.has(column.key),
    );

    // If we have visible columns
    if (filtered.length > 0) {
      const originalRender = filtered[0].render;
      filtered[0] = {
        ...filtered[0],
        headerClassName: "pl-8",
        render: (log: Log) => (
          <div className="flex items-center gap-3 px-2">
            <WarningIcon status={log.response_status} />
            <div className="flex-1 min-w-0">{originalRender(log)}</div>
          </div>
        ),
      };
    }

    return filtered;
  }, [basicColumns, displayProperties]);

  return (
    <VirtualTable
      data={historicalLogs}
      realtimeData={realtimeLogs}
      isLoading={isLoading}
      isFetchingNextPage={isLoadingMore}
      onLoadMore={loadMore}
      columns={visibleColumns}
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
              Keep track of all activity within your workspace. We collect all API requests, giving
              you a clear history to find problems or debug issues.
            </Empty.Description>
            <Empty.Actions className="mt-4 justify-start">
              <a
                href="https://www.unkey.com/docs/introduction"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button size="md">
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
