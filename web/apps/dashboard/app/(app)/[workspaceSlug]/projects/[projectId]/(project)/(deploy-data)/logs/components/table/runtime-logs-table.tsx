"use client";

import {
  type RuntimeLogRow,
  createRuntimeLogsColumns,
  getLogKey,
  getRowClassName,
  getSelectedClassName,
  useRuntimeLogsQuery,
} from "@/components/runtime-logs-table";
import { IconLayers3Outline18 } from "@unkey/icons";
import {
  DataTable,
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  PaginationFooter,
} from "@unkey/ui";
import { useMemo } from "react";
import { useRuntimeLogs } from "../../context/runtime-logs-provider";

export function RuntimeLogsTable() {
  const { selectedLog, setSelectedLog, isLive, refreshNonce } = useRuntimeLogs();
  const {
    realtimeLogs,
    historicalLogs,
    isLoading,
    isCountLoading,
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
      <DataTable<RuntimeLogRow>
        data={historicalLogs}
        realtimeData={realtimeLogs}
        columns={columns}
        getRowId={(log) => log.rowKey ?? getLogKey(log)}
        isLoading={isLoading}
        onRowClick={setSelectedLog}
        selectedItem={selectedLog}
        rowClassName={(log) => getRowClassName(log, selectedLog, isLive, realtimeLogs)}
        selectedClassName={getSelectedClassName}
        emptyState={
          <EmptyState frame="none">
            <EmptyStateIcon>
              <IconLayers3Outline18 />
            </EmptyStateIcon>
            <EmptyStateHeader>
              <EmptyStateTitle>No runtime logs</EmptyStateTitle>
              <EmptyStateDescription>
                No logs found for the selected filters and time range.
              </EmptyStateDescription>
            </EmptyStateHeader>
          </EmptyState>
        }
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
          loading={isLoading || isCountLoading}
          disabled={isNavigating}
        />
      )}
    </div>
  );
}
