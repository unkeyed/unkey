"use client";
import { VerificationLogsTable, createBaseVerificationColumns } from "@/components/logs/verification-table";
import type { EmptyStateConfig } from "@/components/logs/verification-table";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { Button } from "@unkey/ui";
import { BookBookmark } from "@unkey/icons";
import { useCallback, useState } from "react";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { useKeyDetailsLogsContext } from "../../context/logs";
import { useKeyDetailsLogsQuery } from "./hooks/use-logs-query";

const KEY_EMPTY_STATE: EmptyStateConfig = {
  title: "Key Verification Logs",
  description: "No verification logs found for this key. When this API key is used, details about each verification attempt will appear here.",
  actions: (
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
  ),
};

type Props = {
  keyId: string;
  keyspaceId: string;
  selectedLog: KeyDetailsLog | null;
  onLogSelect: (log: KeyDetailsLog | null) => void;
};

export const KeyDetailsLogsTable = ({ keyspaceId, keyId, selectedLog, onLogSelect }: Props) => {
  const { isLive } = useKeyDetailsLogsContext();
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

  const dataHook = () => {
    const { realtimeLogs, historicalLogs, isLoading, isLoadingMore, loadMore, hasMore, totalCount } =
      useKeyDetailsLogsQuery({
        keyId,
        keyspaceId,
        startPolling: isLive,
        pollIntervalMs: 2000,
      });

    return {
      realtimeLogs,
      historicalLogs,
      isLoading,
      isLoadingMore,
      loadMore,
      hasMore: hasMore ?? false,
      totalCount,
    };
  };

  const columns = createBaseVerificationColumns<KeyDetailsLog>(selectedLog);

  return (
    <VerificationLogsTable
      dataHook={dataHook}
      selectedLog={selectedLog}
      onLogSelect={onLogSelect}
      columns={columns}
      emptyState={KEY_EMPTY_STATE}
      onRowMouseEnter={handleRowHover}
      onRowMouseLeave={handleRowMouseLeave}
    />
  );
};