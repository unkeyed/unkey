"use client";
import {
  createKeyDetailsLogsColumns,
  getRowClassName,
  useKeyDetailsLogsQuery,
} from "@/components/key-details-logs-table";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { BookBookmark } from "@unkey/icons";
import { Button, DataTable, Empty, LoadMoreFooter } from "@unkey/ui";
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
  const { realtimeLogs, historicalLogs, isLoading, isLoadingMore, loadMore, hasMore, totalCount } =
    useKeyDetailsLogsQuery({
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

  const columns = useMemo(
    () => createKeyDetailsLogsColumns({ selectedLog }),
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
        config={{ rowHeight: 40, layout: "classic", rowBorders: false }}
        emptyState={
          <div className="w-full flex justify-center items-center h-full">
            <Empty className="w-[400px] flex items-start">
              <Empty.Icon className="w-auto" />
              <Empty.Title>Key Verification Logs</Empty.Title>
              <Empty.Description className="text-left">
                No verification logs found for this key. When this API key is used, details about
                each verification attempt will appear here.
              </Empty.Description>
              <Empty.Actions className="mt-4 justify-center md:justify-start">
                <a
                  href="https://www.unkey.com/docs/introduction"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <Button size="md">
                    <BookBookmark />
                    Documentation
                  </Button>
                </a>
              </Empty.Actions>
            </Empty>
          </div>
        }
      />
      <LoadMoreFooter
        onLoadMore={loadMore}
        isFetchingNextPage={isLoadingMore}
        hasMore={hasMore ?? false}
        hide={isLoading}
        totalVisible={historicalLogs.length + realtimeLogs.length}
        totalCount={totalCount}
        buttonText="Load more logs"
        countInfoText={
          <div className="flex gap-2">
            <span>Showing</span>{" "}
            <span className="text-accent-12">
              {new Intl.NumberFormat().format(historicalLogs.length)}
            </span>
            <span>of</span>
            {new Intl.NumberFormat().format(totalCount)}
            <span>requests</span>
          </div>
        }
      />
    </div>
  );
};
