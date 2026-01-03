"use client";

import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import type { Log } from "@unkey/clickhouse/src/logs";
import { BookBookmark, TriangleWarning2 } from "@unkey/icons";
import { Badge, Button, Empty, TimestampInfo } from "@unkey/ui";
import { useSentinelLogsContext } from "../../context/sentinel-logs-provider";
import { extractResponseField } from "../../utils";
import { useSentinelLogsQuery } from "./hooks/use-sentinel-logs-query";
import {
  WARNING_ICON_STYLES,
  getRowClassName,
  getSelectedClassName,
  getStatusStyle,
} from "./utils/get-row-class";

export const SentinelLogsTable = () => {
  const { setSelectedLog, selectedLog, isLive } = useSentinelLogsContext();
  const { realtimeLogs, historicalLogs, isLoading, isLoadingMore, loadMore, hasMore, total } =
    useSentinelLogsQuery({
      startPolling: isLive,
      pollIntervalMs: 2000,
    });

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
      rowClassName={(log) => getRowClassName({ log, selectedLog, isLive, realtimeLogs })}
      selectedClassName={getSelectedClassName}
      loadMoreFooterProps={{
        hide: isLoading,
        buttonText: "Load more logs",
        hasMore,
        countInfoText: (
          <div className="flex gap-2">
            <span>Showing</span>{" "}
            <span className="text-accent-12">
              {new Intl.NumberFormat().format(historicalLogs.length)}
            </span>
            <span>of</span>
            {new Intl.NumberFormat().format(total)}
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

const WarningIcon = ({ status }: { status: number }) => (
  <TriangleWarning2
    iconSize="md-regular"
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
    width: "180px",
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
    width: "120px",
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
    key: "host",
    header: "Hostname",
    width: "200px",
    render: (log) => (
      <div className="font-mono pr-4 truncate" title={log.host}>
        {log.host}
      </div>
    ),
  },
  {
    key: "method",
    header: "Method",
    width: "80px",
    render: (log) => (
      <Badge
        className={cn(
          "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
          getStatusStyle(log.response_status).badge.default,
        )}
      >
        {log.method}
      </Badge>
    ),
  },
  {
    key: "path",
    header: "Path",
    width: "250px",
    render: (log) => (
      <div className="font-mono pr-4 truncate" title={log.path}>
        {log.path}
      </div>
    ),
  },
  {
    key: "response_body",
    header: "Response Body",
    width: "300px",
    render: (log) => (
      <div className="font-mono whitespace-nowrap truncate max-w-[300px]" title={log.response_body}>
        {log.response_body}
      </div>
    ),
  },
  {
    key: "request_body",
    header: "Request Body",
    width: "1fr",
    render: (log) => (
      <div className="font-mono whitespace-nowrap truncate max-w-[300px]" title={log.request_body}>
        {log.request_body}
      </div>
    ),
  },
];
