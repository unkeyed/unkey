"use client";
import { EmptyOrLoadError } from "@/components/load-error";

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
} from "@unkey/ui";
import { useMemo } from "react";
import { useRuntimeLogs } from "../../context/runtime-logs-provider";

export function RuntimeLogsTable() {
  const { selectedLog, setSelectedLog, isLive, refreshNonce } = useRuntimeLogs();
  const { realtimeLogs, table, totalCount } = useRuntimeLogsQuery({
    startPolling: isLive,
    refreshNonce,
  });

  const selectedLogKey = selectedLog ? getLogKey(selectedLog) : null;
  const columns = useMemo(() => createRuntimeLogsColumns({ selectedLogKey }), [selectedLogKey]);

  return (
    <div className="flex flex-col md:flex-1">
      <DataTable<RuntimeLogRow>
        data={table.rows}
        realtimeData={realtimeLogs}
        columns={columns}
        getRowId={(log) => log.rowKey ?? getLogKey(log)}
        isLoading={table.isLoading}
        loadMore={{
          singular: "log",
          plural: "logs",
          state: table.footer,
          totalCount,
        }}
        onRowClick={setSelectedLog}
        selectedItem={selectedLog}
        rowClassName={(log) => getRowClassName(log, selectedLog, isLive, realtimeLogs)}
        selectedClassName={getSelectedClassName}
        emptyState={
          <EmptyOrLoadError failure={table.failure} title="Could not load runtime logs">
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
          </EmptyOrLoadError>
        }
        // VirtualTable defaulted to 26px dense rows and 50 loading rows; DataTable's
        // defaults (36px / 10) differ, so set them explicitly to preserve the layout.
        config={{
          rowHeight: 26,
          layout: "classic",
          rowBorders: false,
          loadingRows: 50,
        }}
      />
    </div>
  );
}
