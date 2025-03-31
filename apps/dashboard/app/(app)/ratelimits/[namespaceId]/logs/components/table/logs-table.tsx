"use client";

import { safeParseJson } from "@/app/(app)/logs/utils";
import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import type { RatelimitLog } from "@unkey/clickhouse/src/ratelimits";
import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useMemo } from "react";
import { DEFAULT_STATUS_FLAG } from "../../constants";
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
};

const getStatusStyle = (rejected: number): StatusStyle => {
  if (rejected === DEFAULT_STATUS_FLAG) {
    return STATUS_STYLES.warning;
  }
  return STATUS_STYLES.success;
};

const getSelectedClassName = (log: RatelimitLog, isSelected: boolean) => {
  if (!isSelected) {
    return "";
  }
  const style = getStatusStyle(log.status);
  return style.selected;
};

export const RatelimitLogsTable = () => {
  const { setSelectedLog, selectedLog, isLive, namespaceId } = useRatelimitLogsContext();
  const { realtimeLogs, historicalLogs, isLoading, isLoadingMore, loadMore } =
    useRatelimitLogsQuery({
      namespaceId,
      startPolling: isLive,
      pollIntervalMs: 2000,
    });

  const getRowClassName = (log: RatelimitLog) => {
    const style = getStatusStyle(log.status);
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
        headerClassName: "pl-2",
        width: "10%",
        render: (log) => (
          <div className="flex items-center gap-3 pl-2 truncate mr-3">
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
        render: (log) => <div className="font-mono truncate mr-1 max-w-40">{log.identifier}</div>,
      },
      {
        key: "rejected",
        header: "Status",
        width: "10%",
        render: (log) => {
          const style = getStatusStyle(log.status);
          const isSelected = selectedLog?.request_id === log.request_id;
          return (
            <Badge
              className={cn(
                "uppercase px-[6px] rounded-md font-mono min-w-[70px] inline-block text-center",
                isSelected ? style.badge.selected : style.badge.default,
              )}
            >
              {log.status === 0 ? "Blocked" : "Passed"}
            </Badge>
          );
        },
      },
      {
        key: "limit",
        header: "Limit",
        width: "auto",
        render: (log) => {
          return (
            <div className="font-mono">{safeParseJson(log.response_body)?.limit ?? "<EMPTY>"}</div>
          );
        },
      },
      {
        key: "duration",
        header: "Duration",
        width: "auto",
        render: (log) => {
          const parsedDuration = safeParseJson(log.request_body)?.duration;
          return (
            <div className="font-mono">
              {parsedDuration ? msToSeconds(parsedDuration) : "<EMPTY>"}
            </div>
          );
        },
      },
      {
        key: "reset",
        header: "Resets At",
        width: "auto",
        render: (log) => {
          const parsedReset = safeParseJson(log.response_body)?.reset;
          return parsedReset ? (
            <div className="font-mono">
              <TimestampInfo
                value={safeParseJson(log.response_body)?.reset ?? "<EMPTY>"}
                className={cn(
                  "font-mono group-hover:underline decoration-dotted",
                  selectedLog && selectedLog.request_id !== log.request_id && "pointer-events-none",
                )}
              />
            </div>
          ) : (
            "<Empty>"
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
        key: "actions",
        header: "",
        width: "auto",
        render: (log) => (
          <div className="text-end">
            <LogsTableAction identifier={log.identifier} />
          </div>
        ),
      },
    ],
    [selectedLog?.request_id],
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
              No ratelimit logs yet. Once API requests start coming in, you'll see a detailed view
              of your rate limits, including passed and blocked requests, across your API endpoints.
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

function msToSeconds(ms: number) {
  const seconds = Math.round(ms / 1000);
  return `${seconds}s`;
}
