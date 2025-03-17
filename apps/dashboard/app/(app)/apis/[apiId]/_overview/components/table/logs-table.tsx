"use client";
import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { Ban, BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";

import { formatNumber } from "@/lib/fmt";
import { OutcomesPopover } from "./components/outcome-popover";
import { KeyIdentifierColumn } from "./components/override-indicator";
import { useKeysOverviewLogsQuery } from "./hooks/use-logs-query";
import { getErrorPercentage, getSuccessPercentage } from "./utils/calculate-blocked-percentage";
import { STATUS_STYLES, getRowClassName, getStatusStyle } from "./utils/get-row-class";

type Props = {
  log: KeysOverviewLog | null;
  setSelectedLog: (data: KeysOverviewLog | null) => void;
  apiId: string;
};

export const KeysOverviewLogsTable = ({ apiId, setSelectedLog, log: selectedLog }: Props) => {
  const { historicalLogs, isLoading, isLoadingMore, loadMore } = useKeysOverviewLogsQuery({
    apiId,
  });

  const columns = (): Column<KeysOverviewLog>[] => {
    return [
      {
        key: "key_id",
        header: "ID",
        width: "15%",
        headerClassName: "pl-11",
        render: (log) => <KeyIdentifierColumn log={log} />,
      },
      {
        key: "name",
        header: "Name",
        width: "15%",
        render: (log) => {
          const name = log.key_details?.name || "—";
          return (
            <div className="flex items-center font-mono">
              <div className="w-full max-w-[150px] truncate whitespace-nowrap" title={name}>
                {name}
              </div>
            </div>
          );
        },
      },
      {
        key: "external_id",
        header: "External ID",
        width: "15%",
        render: (log) => {
          const externalId =
            (log.key_details?.identity?.external_id ?? log.key_details?.owner_id) || "—";
          return (
            <div className="flex items-center font-mono">
              <div className="w-full max-w-[150px] truncate whitespace-nowrap" title={externalId}>
                {externalId}
              </div>
            </div>
          );
        },
      },
      {
        key: "valid",
        header: "Valid",
        width: "15%",
        render: (log) => {
          const successPercentage = getSuccessPercentage(log);
          return (
            <div className="flex gap-3 items-center tabular-nums">
              <Badge
                className={cn(
                  "px-[6px] rounded-md font-mono whitespace-nowrap",
                  selectedLog?.key_id === log.key_id
                    ? STATUS_STYLES.success.badge.selected
                    : STATUS_STYLES.success.badge.default,
                )}
                title={`${log.valid_count.toLocaleString()} Valid requests (${successPercentage.toFixed(
                  1,
                )}%)`}
              >
                {formatNumber(log.valid_count)}
              </Badge>
            </div>
          );
        },
      },
      {
        key: "invalid",
        header: "Invalid",
        width: "15%",
        render: (log) => {
          const style = getStatusStyle(log);
          const errorPercentage = getErrorPercentage(log);

          return (
            <div className="flex items-center w-full">
              <div className="flex-shrink-0">
                <Badge
                  className={cn(
                    "px-[6px] rounded-md font-mono whitespace-nowrap flex items-center",
                    selectedLog?.key_id === log.key_id ? style.badge.selected : style.badge.default,
                  )}
                  title={`${log.error_count.toLocaleString()} Invalid requests (${errorPercentage.toFixed(
                    1,
                  )}%)`}
                >
                  <span className="mr-[6px] flex-shrink-0">
                    <Ban size="sm-regular" />
                  </span>
                  <span className="overflow-hidden text-ellipsis whitespace-nowrap max-w-[45px]">
                    {formatNumber(log.error_count)}
                  </span>
                </Badge>
              </div>
              <div className="ml-2 flex-shrink-0">
                <OutcomesPopover
                  outcomeCounts={log.outcome_counts}
                  isSelected={selectedLog?.key_id === log.key_id}
                />
              </div>
            </div>
          );
        },
      },
      {
        key: "lastUsed",
        header: "Last Used",
        width: "15%",
        render: (log) => (
          <TimestampInfo
            value={log.time}
            className={cn(
              "font-mono group-hover:underline decoration-dotted",
              selectedLog && selectedLog.request_id !== log.request_id && "pointer-events-none",
            )}
          />
        ),
      },
    ];
  };

  return (
    <VirtualTable
      data={historicalLogs}
      isLoading={isLoading}
      isFetchingNextPage={isLoadingMore}
      onLoadMore={loadMore}
      columns={columns()}
      onRowClick={setSelectedLog}
      selectedItem={selectedLog}
      keyExtractor={(log) => log.request_id}
      rowClassName={(log) => getRowClassName(log, selectedLog as KeysOverviewLog)}
      emptyState={
        <div className="w-full flex justify-center items-center h-full">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>Key Verification Logs</Empty.Title>
            <Empty.Description className="text-left">
              No key verification data to show. Once requests are made with API keys, you'll see a
              summary of successful and failed verification attempts.
            </Empty.Description>
            <Empty.Actions className="mt-4 justify-start">
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
  );
};
