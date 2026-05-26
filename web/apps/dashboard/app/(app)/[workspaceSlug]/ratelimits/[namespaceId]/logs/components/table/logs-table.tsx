"use client";

import {
  EmptyRatelimitLogs,
  createRatelimitLogsColumns,
  getRowClassName,
  useRatelimitLogsQuery,
} from "@/components/ratelimit-logs-table";
import type { RowSelectionState } from "@tanstack/react-table";
import { DataTable, PaginationFooter } from "@unkey/ui";
import { useMemo } from "react";
import { useRatelimitLogsContext } from "../../context/logs";

export const RatelimitLogsTable = () => {
  const { setSelectedLog, selectedLog, isLive, namespaceId } = useRatelimitLogsContext();
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
    sorting,
    onSortingChange,
  } = useRatelimitLogsQuery({
    namespaceId,
    startPolling: isLive,
    pollIntervalMs: 2000,
  });

  // Sorting is server-side and only applies to historical pages, so it's
  // disabled while the live tail is running.
  const columns = useMemo(() => createRatelimitLogsColumns({ enableSorting: !isLive }), [isLive]);

  const rowSelection = useMemo<RowSelectionState>(
    () => (selectedLog ? { [selectedLog.request_id]: true } : {}),
    [selectedLog],
  );

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
        rowClassName={(log) => getRowClassName(log, selectedLog)}
        enableRowSelection={true}
        rowSelection={rowSelection}
        sorting={sorting}
        onSortingChange={onSortingChange}
        manualSorting
        config={{ rowHeight: 26, layout: "classic", rowBorders: false, containerPadding: "px-2" }}
        emptyState={<EmptyRatelimitLogs />}
      />
      {!isLive && totalPages > 1 && (
        <PaginationFooter
          page={page}
          pageSize={pageSize}
          totalPages={totalPages}
          totalCount={totalCount}
          onPageChange={onPageChange}
          itemLabel="ratelimit requests"
          loading={isLoading}
          disabled={isNavigating}
        />
      )}
    </div>
  );
};
