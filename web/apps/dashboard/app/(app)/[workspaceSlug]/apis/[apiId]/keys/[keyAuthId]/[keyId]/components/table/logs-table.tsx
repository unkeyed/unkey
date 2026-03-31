"use client";
import {
  EmptyKeyDetailsLogs,
  createKeyDetailsLogsColumns,
  getRowClassName,
  useKeyDetailsLogsQuery,
} from "@/components/key-details-logs-table";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import type { RowSelectionState } from "@tanstack/react-table";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { DataTable, PaginationFooter } from "@unkey/ui";
import { useCallback, useMemo, useState } from "react";
import { useKeyDetailsLogsContext } from "../../context/logs";

type Props = {
  keyId: string;
  keyspaceId: string;
  selectedLog: KeyDetailsLog | null;
  onLogSelect: (log: KeyDetailsLog | null) => void;
};

export const KeyDetailsLogsTable = ({ keyspaceId, keyId, selectedLog, onLogSelect }: Props) => {
  const { isLive } = useKeyDetailsLogsContext();
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
  } = useKeyDetailsLogsQuery({
    keyId,
    keyspaceId,
    startPolling: isLive,
    pollIntervalMs: 2000,
  });

  const [hoveredLogId, setHoveredLogId] = useState<string | null>(null);
  const { queryTime: timestamp } = useQueryTime();
  const utils = trpc.useUtils();

  const handleRowHover = useCallback(
    (log: KeyDetailsLog) => {
      if (log.request_id !== hoveredLogId) {
        setHoveredLogId(log.request_id);

        utils.logs.queryLogs.prefetch({
          limit: 1,
          startTime: 0,
          endTime: timestamp,
          host: { filters: [] },
          method: { filters: [] },
          path: { filters: [] },
          status: { filters: [] },
          requestId: {
            filters: [
              {
                operator: "is",
                value: log.request_id,
              },
            ],
          },
          since: "",
        });
      }
    },
    [hoveredLogId, utils.logs.queryLogs, timestamp],
  );

  const handleRowMouseLeave = useCallback(() => {
    setHoveredLogId(null);
  }, []);

  const columns = useMemo(() => createKeyDetailsLogsColumns(), []);

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
        onRowClick={onLogSelect}
        onRowMouseEnter={handleRowHover}
        onRowMouseLeave={handleRowMouseLeave}
        selectedItem={selectedLog}
        rowClassName={(log) => getRowClassName(log, selectedLog)}
        enableRowSelection={true}
        rowSelection={rowSelection}
        config={{ rowHeight: 26, layout: "classic", rowBorders: false }}
        emptyState={<EmptyKeyDetailsLogs />}
      />
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
    </div>
  );
};
