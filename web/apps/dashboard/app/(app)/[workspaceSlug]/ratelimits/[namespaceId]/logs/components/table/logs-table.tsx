"use client";

import {
  EmptyRatelimitLogs,
  type EnrichedRatelimitLog,
  createRatelimitLogsColumns,
  getRowClassName,
  renderRatelimitLogsSkeletonRow,
  useRatelimitLogsQuery,
} from "@/components/ratelimit-logs-table";
import { cn } from "@/lib/utils";
import type { RowSelectionState } from "@tanstack/react-table";
import { DataTable, PaginationFooter } from "@unkey/ui";
import { useCallback, useMemo } from "react";
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

  const rowClassName = useCallback(
    (log: EnrichedRatelimitLog) =>
      cn(
        getRowClassName(log, selectedLog),
        // During live tail, dim rows that have already aged into the historical
        // set so the newest streaming rows stand out; hovering restores full
        // opacity. Lives here, not in getRowClassName, because only the consumer
        // knows the live-tail and realtime state.
        isLive &&
          !realtimeLogs.some((realtime) => realtime.request_id === log.request_id) && [
            "opacity-50",
            "hover:opacity-100",
          ],
        // When a row is selected, fade and drop the other rows behind it so the
        // open detail row reads as the focus.
        selectedLog && {
          "opacity-50 z-0": selectedLog.request_id !== log.request_id,
          "opacity-100 z-10": selectedLog.request_id === log.request_id,
        },
      ),
    [isLive, realtimeLogs, selectedLog],
  );

  return (
    <div className="flex flex-col">
      <DataTable
        data={historicalLogs}
        realtimeData={realtimeLogs}
        columns={columns}
        getRowId={(log) => log.request_id}
        isLoading={isLoading}
        renderSkeletonRow={renderRatelimitLogsSkeletonRow}
        onRowClick={setSelectedLog}
        selectedItem={selectedLog}
        rowClassName={rowClassName}
        enableRowSelection={true}
        rowSelection={rowSelection}
        sorting={sorting}
        onSortingChange={onSortingChange}
        manualSorting
        config={{ rowHeight: 26, layout: "classic", rowBorders: false, containerPadding: "px-2" }}
        emptyState={<EmptyRatelimitLogs />}
      />
      {!isLive && (
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
