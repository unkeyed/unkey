"use client";

import {
  createSentinelLogsColumns,
  getRowClassName,
  getSelectedClassName,
  useSentinelLogsQuery,
} from "@/components/sentinel-logs-table";
import { BookBookmark } from "@unkey/icons";
import { Button, DataTable, Empty, PaginationFooter } from "@unkey/ui";
import { useMemo } from "react";
import { useSentinelLogsContext } from "../../context/sentinel-logs-provider";

export const SentinelLogsTable = () => {
  const { setSelectedLog, selectedLog, isLive, refreshNonce } = useSentinelLogsContext();
  const {
    realtimeLogs,
    historicalLogs,
    isLoading,
    isNavigating,
    totalCount,
    page,
    pageSize,
    totalPages,
    onPageChange,
  } = useSentinelLogsQuery({
    startPolling: isLive,
    pollIntervalMs: 2000,
    refreshNonce,
  });

  const columns = useMemo(() => createSentinelLogsColumns(), []);

  return (
    <div className="flex flex-col">
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
        emptyState={<EmptyState />}
        // VirtualTable defaulted to 26px dense rows and 50 loading rows; DataTable's
        // defaults (36px / 10) differ, so set them explicitly to preserve the layout.
        config={{ rowHeight: 26, layout: "classic", rowBorders: false, loadingRows: 50 }}
      />
      {/* Live mode is a streaming view of the newest logs (page 1 only), matching
          the other live log tables — pagination is hidden while live. */}
      {!isLive && (
        <PaginationFooter
          page={page}
          pageSize={pageSize}
          totalPages={totalPages}
          totalCount={totalCount}
          onPageChange={onPageChange}
          itemLabel="requests"
          loading={isLoading}
          disabled={isNavigating}
        />
      )}
    </div>
  );
};

const EmptyState = () => (
  <div className="w-full flex justify-center items-center h-full">
    <Empty className="w-100 flex items-start">
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
