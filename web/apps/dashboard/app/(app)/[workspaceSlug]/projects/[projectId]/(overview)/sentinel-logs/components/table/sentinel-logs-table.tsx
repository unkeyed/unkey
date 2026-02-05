"use client";

import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { formatNumber } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import type { SentinelLogsResponse } from "@unkey/clickhouse/src/sentinel";
import { BookBookmark, TriangleWarning2 } from "@unkey/icons";
import { Badge, Button, Empty, TimestampInfo } from "@unkey/ui";
import { useSentinelLogsContext } from "../../context/sentinel-logs-provider";
import { useSentinelLogsQuery } from "./hooks/use-sentinel-logs-query";
import {
  WARNING_ICON_STYLES,
  getRowClassName,
  getSelectedClassName,
  getStatusStyle,
} from "./utils/get-row-class";

export const SentinelLogsTable = () => {
  const { setSelectedLog, selectedLog } = useSentinelLogsContext();
  const { logs, isLoading, isLoadingMore, loadMore, hasMore, total } = useSentinelLogsQuery();

  return (
    <VirtualTable
      data={logs}
      isLoading={isLoading}
      isFetchingNextPage={isLoadingMore}
      onLoadMore={loadMore}
      columns={columns}
      onRowClick={setSelectedLog}
      selectedItem={selectedLog}
      keyExtractor={(log) => log.request_id}
      rowClassName={(log) => getRowClassName(log, selectedLog)}
      selectedClassName={getSelectedClassName}
      loadMoreFooterProps={{
        hide: isLoading,
        buttonText: "Load more logs",
        hasMore,
        countInfoText: (
          <div className="flex gap-2">
            <span>Showing</span>
            <span className="text-accent-12">{formatNumber(logs.length)}</span>
            <span>of</span>
            {formatNumber(total)}
            <span>requests</span>
          </div>
        ),
      }}
      emptyState={<EmptyState />}
    />
  );
};

const EmptyState = () => (
  <div className="w-full flex justify-center items-center h-full">
    <Empty className="w-[400px] flex items-start">
      <Empty.Icon className="w-auto" />
      <Empty.Title>Logs</Empty.Title>
      <Empty.Description className="text-left">
        Keep track of all activity within your workspace. We collect all API requests, giving you a
        clear history to find problems or debug issues.
      </Empty.Description>
      <Empty.Actions className="mt-4 justify-start">
        <a href="https://www.unkey.com/docs/introduction" target="_blank" rel="noopener noreferrer">
          <Button size="md">
            <BookBookmark />
            Documentation
          </Button>
        </a>
      </Empty.Actions>
    </Empty>
  </div>
);

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

const columns: Column<SentinelLogsResponse>[] = [
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
          {log.response_status}
        </Badge>
      );
    },
  },
  {
    key: "region",
    header: "Region",
    width: "100px",
    render: (log) => (
      <div className="font-mono pr-4 truncate" title={log.region}>
        {log.region}
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
    key: "total_latency",
    header: "Total Latency",
    width: "120px",
    render: (log) => <div className="font-mono pr-4">{log.total_latency}ms</div>,
  },
  {
    key: "deployment_id",
    header: "Deployment ID",
    width: "1fr",
    render: (log) => (
      <div className="font-mono whitespace-nowrap truncate max-w-[200px]" title={log.deployment_id}>
        {log.deployment_id}
      </div>
    ),
  },
];
