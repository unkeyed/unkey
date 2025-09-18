"use client";

import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import type { Log } from "@unkey/clickhouse/src/logs";
import { BookBookmark, TriangleWarning2 } from "@unkey/icons";
import { Badge, Button, Empty, TimestampInfo } from "@unkey/ui";
import { useGatewayLogsContext } from "../../context/gateway-logs-provider";
import { extractResponseField } from "../../utils";
import { useGatewayLogsQuery } from "./hooks/use-gateway-logs-query";

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
    size="md-regular"
    className={cn(
      WARNING_ICON_STYLES.base,
      status < 300 && "invisible",
      status >= 400 && status < 500 && WARNING_ICON_STYLES.warning,
      status >= 500 && WARNING_ICON_STYLES.error,
    )}
  />
);

const columns: Column<Log>[] = [
  {
    key: "time",
    header: "Time",
    width: "5%",
    headerClassName: "pl-8",
    render: (log) => (
      <div className="flex items-center gap-3 px-2">
        <WarningIcon status={log.response_status} />
        <div className="flex-1 min-w-0">
          <TimestampInfo
            value={log.time}
            className="font-mono group-hover:underline decoration-dotted"
          />
        </div>
      </div>
    ),
  },
  {
    key: "response_status",
    header: "Status",
    width: "7.5%",
    render: (log) => {
      const style = getStatusStyle(log.response_status);
      return (
        <Badge
          className={cn(
            "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
            style.badge.default,
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
    render: (log) => (
      <Badge
        className={cn(
          "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
          STATUS_STYLES.success.badge.default,
        )}
      >
        {log.method}
      </Badge>
    ),
  },
  {
    key: "path",
    header: "Path",
    width: "15%",
    render: (log) => <div className="font-mono pr-4">{log.path}</div>,
  },
  {
    key: "response_body",
    header: "Response Body",
    width: "auto",
    render: (log) => (
      <div className="font-mono whitespace-nowrap truncate w-[500px]">{log.response_body}</div>
    ),
  },
  {
    key: "request_body",
    header: "Request Body",
    width: "auto",
    render: (log) => (
      <div className="font-mono whitespace-nowrap truncate w-[500px]">{log.request_body}</div>
    ),
  },
];

export const GatewayLogsTable = () => {
  const { setSelectedLog, selectedLog, isLive } = useGatewayLogsContext();
  const { realtimeLogs, historicalLogs, isLoading, isLoadingMore, loadMore, hasMore, total } =
    useGatewayLogsQuery({
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
      loadMoreFooterProps={{
        hide: isLoading,
        buttonText: "Load more logs",
        hasMore,
        countInfoText: (
          <div className="flex gap-2">
            <span>Showing</span> <span className="text-accent-12">{historicalLogs.length}</span>
            <span>of</span>
            {total}
            <span>requests</span>
          </div>
        ),
      }}
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
