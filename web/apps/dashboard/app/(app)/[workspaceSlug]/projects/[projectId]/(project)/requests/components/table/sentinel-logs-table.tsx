"use client";

import {
  createSentinelLogsColumns,
  getRowClassName,
  getSelectedClassName,
  useSentinelLogsQuery,
} from "@/components/sentinel-logs-table";
import { formatNumber } from "@/lib/fmt";
import { BookBookmark } from "@unkey/icons";
import { Button, DataTable, Empty } from "@unkey/ui";
import { useMemo } from "react";
import { useSentinelLogsContext } from "../../context/sentinel-logs-provider";

export const SentinelLogsTable = () => {
  const { setSelectedLog, selectedLog, isLive } = useSentinelLogsContext();
  const { realtimeLogs, historicalLogs, isLoading, isLoadingMore, loadMore, hasMore, total } =
    useSentinelLogsQuery({
      startPolling: isLive,
      pollIntervalMs: 2000,
    });

  const columns = useMemo(() => createSentinelLogsColumns(), []);

  return (
    <DataTable
      data={historicalLogs}
      realtimeData={realtimeLogs}
      columns={columns}
      getRowId={(log) => log.request_id}
      isLoading={isLoading}
      onRowClick={setSelectedLog}
      selectedItem={selectedLog}
      rowClassName={(log) => getRowClassName(log, selectedLog, isLive, realtimeLogs)}
      selectedClassName={getSelectedClassName}
      loadMoreFooterProps={{
        hide: isLoading,
        buttonText: "Load more logs",
        hasMore: hasMore ?? false,
        onLoadMore: loadMore,
        isFetchingNextPage: isLoadingMore,
        countInfoText: (
          <div className="flex gap-2">
            <span>Showing</span>
            <span className="text-accent-12">{formatNumber(historicalLogs.length)}</span>
            <span>of</span>
            {formatNumber(total)}
            <span>requests</span>
          </div>
        ),
      }}
      emptyState={<EmptyState />}
      // VirtualTable defaulted to 26px dense rows and 50 loading rows; DataTable's
      // defaults (36px / 10) differ, so set them explicitly to preserve the layout.
      config={{ rowHeight: 26, layout: "classic", rowBorders: false, loadingRows: 50 }}
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
