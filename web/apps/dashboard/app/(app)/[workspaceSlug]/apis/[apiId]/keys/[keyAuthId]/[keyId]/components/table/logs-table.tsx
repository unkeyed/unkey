"use client";
import {
  EmptyKeyDetailsLogs,
  buildGatewayRequestDetailsQueryParams,
  buildRequestDetailsQueryParams,
  createKeyDetailsLogsColumns,
  getRowClassName,
  useKeyDetailsLogsQuery,
} from "@/components/key-details-logs-table";
import { trpc } from "@/lib/trpc/client";
import type { RowSelectionState } from "@tanstack/react-table";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { DataTable, PaginationFooter } from "@unkey/ui";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
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
  const utils = trpc.useUtils();
  const hoverTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Clear any pending hover-prefetch timeout on unmount
  useEffect(() => {
    return () => {
      if (hoverTimerRef.current) {
        clearTimeout(hoverTimerRef.current);
        hoverTimerRef.current = null;
      }
    };
  }, []);

  const handleRowHover = useCallback(
    (log: KeyDetailsLog) => {
      if (log.request_id !== hoveredLogId) {
        setHoveredLogId(log.request_id);

        // Debounce the prefetch so rapid mouse movement across rows
        // doesn't fire N separate queryLogs API calls.
        if (hoverTimerRef.current) {
          clearTimeout(hoverTimerRef.current);
        }
        hoverTimerRef.current = setTimeout(() => {
          const target = { requestId: log.request_id, time: log.time };
          if (log.source === "gateway") {
            utils.deploy.requestLogs.details.prefetch(
              buildGatewayRequestDetailsQueryParams(target),
              { staleTime: Number.POSITIVE_INFINITY },
            );
            return;
          }
          utils.logs.queryLogs.prefetch(buildRequestDetailsQueryParams(target), {
            staleTime: Number.POSITIVE_INFINITY,
          });
        }, 150);
      }
    },
    [hoveredLogId, utils.logs.queryLogs, utils.deploy.requestLogs.details],
  );

  const handleRowMouseLeave = useCallback(() => {
    setHoveredLogId(null);
    if (hoverTimerRef.current) {
      clearTimeout(hoverTimerRef.current);
      hoverTimerRef.current = null;
    }
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
