"use client";

import {
  createRuntimeLogsColumns,
  getLogKey,
  getRowClassName,
  getSelectedClassName,
  useRuntimeLogsQuery,
} from "@/components/runtime-logs-table";
import { DataTable, Empty, PaginationFooter } from "@unkey/ui";
import { useMemo } from "react";
import { useRuntimeLogs } from "../../context/runtime-logs-provider";

export function RuntimeLogsTable() {
  const { selectedLog, setSelectedLog, isLive, refreshNonce } = useRuntimeLogs();
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
  } = useRuntimeLogsQuery({ startPolling: isLive, refreshNonce });

  const selectedLogKey = selectedLog ? getLogKey(selectedLog) : null;
  const columns = useMemo(() => createRuntimeLogsColumns({ selectedLogKey }), [selectedLogKey]);

  return (
    <div className="flex flex-col">
      <DataTable
        data={historicalLogs}
        realtimeData={realtimeLogs}
        columns={columns}
        getRowId={getLogKey}
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
          itemLabel="logs"
          loading={isLoading}
          disabled={isNavigating}
        />
      )}
    </div>
  );
}

const EmptyState = () => (
  <div className="w-full flex justify-center items-center h-full">
    <Empty className="w-100 flex items-start">
      <Empty.Icon className="w-auto" />
      <Empty.Title>No runtime logs</Empty.Title>
      <Empty.Description className="text-left">
        No logs found for the selected filters and time range.
      </Empty.Description>
    </Empty>
  </div>
);
