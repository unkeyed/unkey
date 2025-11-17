"use client";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { useQueryTime } from "@/providers/query-time-provider";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { TimestampInfo } from "@unkey/ui";
import { useCallback, useMemo, useState } from "react";
import { useKeyDetailsLogsContext } from "../../context/logs";
import { BadgeList } from "./components/badge-list";
import { EmptyTable } from "./components/empty-table";
import { StatusCell, getStatusStyle } from "./components/status-cell";
import { useKeyDetailsLogsQuery } from "./hooks/use-logs-query";

type Props = {
  keyId: string;
  keyspaceId: string;
  selectedLog: KeyDetailsLog | null;
  onLogSelect: (log: KeyDetailsLog | null) => void;
};

const POLL_INTERVAL_MS = 2000;
const PREFETCH_LIMIT = 1;
const PREFETCH_START_TIME = 0;

export const KeyDetailsLogsTable = ({ keyspaceId, keyId, selectedLog, onLogSelect }: Props) => {
  const { isLive } = useKeyDetailsLogsContext();
  const { realtimeLogs, historicalLogs, isLoading, isLoadingMore, loadMore, hasMore, totalCount } =
    useKeyDetailsLogsQuery({
      keyId,
      keyspaceId,
      startPolling: isLive,
      pollIntervalMs: POLL_INTERVAL_MS,
    });

  const getRowClassName = useCallback(
    (log: KeyDetailsLog) => {
      const style = getStatusStyle(log);
      const isSelected = selectedLog?.request_id === log.request_id;
      return cn(
        style.base,
        style.hover,
        "group rounded-md cursor-pointer transition-colors",
        "focus:outline-none focus:ring-1 focus:ring-opacity-40",
        style.focusRing,
        isSelected && style.selected,
      );
    },
    [selectedLog],
  );

  const [hoveredLogId, setHoveredLogId] = useState<string | null>(null);
  const { queryTime: timestamp } = useQueryTime();
  const utils = trpc.useUtils();

  const handleRowHover = useCallback(
    (log: KeyDetailsLog) => {
      if (log.request_id !== hoveredLogId) {
        setHoveredLogId(log.request_id);

        utils.logs.queryLogs.prefetch({
          limit: PREFETCH_LIMIT,
          startTime: PREFETCH_START_TIME,
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

  const columns = useMemo<Column<KeyDetailsLog>[]>(
    () => [
      {
        key: "time",
        header: "Time",
        width: "15%",
        headerClassName: "pl-2",
        render: (log) => (
          <TimestampInfo
            value={log.time}
            className={cn(
              "font-mono group-hover:underline decoration-dotted pl-2",
              selectedLog && selectedLog.request_id !== log.request_id && "pointer-events-none",
            )}
          />
        ),
      },
      {
        key: "outcome",
        header: "Status",
        width: "15%",
        render: (log) => <StatusCell log={log} selectedLog={selectedLog} />,
      },
      {
        key: "region",
        header: "Region",
        width: "15%",
        render: (log) => (
          <div className="flex items-center font-mono">
            <div className="w-full whitespace-nowrap" title={log.region}>
              {log.region}
            </div>
          </div>
        ),
      },
      {
        key: "tags",
        header: "Tags",
        width: "auto",
        render: (log) => <BadgeList log={log} selectedLog={selectedLog} />,
      },
    ],
    [selectedLog],
  );

  return (
    <VirtualTable
      data={historicalLogs}
      realtimeData={realtimeLogs}
      isLoading={isLoading}
      isFetchingNextPage={isLoadingMore}
      onLoadMore={loadMore}
      columns={columns}
      onRowClick={onLogSelect}
      onRowMouseEnter={handleRowHover}
      onRowMouseLeave={handleRowMouseLeave}
      selectedItem={selectedLog}
      keyExtractor={(log) => log.request_id}
      rowClassName={getRowClassName}
      loadMoreFooterProps={{
        buttonText: "Load more logs",
        hasMore,
        hide: isLoading,
        countInfoText: (
          <div className="flex gap-2">
            <span>Showing</span>{" "}
            <span className="text-accent-12">
              {new Intl.NumberFormat().format(historicalLogs.length)}
            </span>
            <span>of</span>
            {new Intl.NumberFormat().format(totalCount)}
            <span>requests</span>
          </div>
        ),
      }}
      emptyState={
        <EmptyTable
          title="Key Verification Logs"
          description="No verification logs found for this key. When this API key is used, details about each verification attempt will appear here."
        />
      }
    />
  );
};
