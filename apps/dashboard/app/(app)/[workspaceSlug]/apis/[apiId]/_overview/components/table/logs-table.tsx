"use client";
import { cn } from "@/lib/utils";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { Ban, BookBookmark } from "@unkey/icons";
import { Badge, Button, Empty, Table, TimestampInfo, type ColumnDef } from "@unkey/ui";

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

  const columns: ColumnDef<KeysOverviewLog>[] = [
    {
      id: "key_id",
      header: "ID",
      size: 150,
      cell: ({ row }) => (
        <KeyIdentifierColumn
          log={row.original}
          apiId={apiId}
          onNavigate={() => setSelectedLog(null)}
        />
      ),
    },
    {
      id: "name",
      header: "Name",
      size: 150,
      cell: ({ row }) => {
        const name = row.original.key_details?.name || "—";
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
      id: "external_id",
      header: "External ID",
      size: 150,
      cell: ({ row }) => {
        const externalId =
          (row.original.key_details?.identity?.external_id ?? row.original.key_details?.owner_id) ||
          "—";
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
      id: "valid",
      header: "Valid",
      size: 150,
      enableSorting: true,
      cell: ({ row }) => {
        const successPercentage = getSuccessPercentage(row.original);
        return (
          <div className="flex gap-3 items-center tabular-nums">
            <Badge
              className={cn(
                "px-[6px] rounded-md font-mono whitespace-nowrap",
                selectedLog?.key_id === row.original.key_id
                  ? STATUS_STYLES.success.badge.selected
                  : STATUS_STYLES.success.badge.default,
              )}
              title={`${row.original.valid_count.toLocaleString()} Valid requests (${successPercentage.toFixed(
                1,
              )}%)`}
            >
              {formatNumber(row.original.valid_count)}
            </Badge>
          </div>
        );
      },
    },
    {
      id: "invalid",
      header: "Invalid",
      size: 150,
      enableSorting: true,
      cell: ({ row }) => {
        const style = getStatusStyle(row.original);
        const errorPercentage = getErrorPercentage(row.original);

        return (
          <div className="flex items-center w-full">
            <div className="flex-shrink-0">
              <Badge
                className={cn(
                  "px-[6px] rounded-md font-mono whitespace-nowrap flex items-center",
                  selectedLog?.key_id === row.original.key_id
                    ? style.badge.selected
                    : style.badge.default,
                )}
                title={`${row.original.error_count.toLocaleString()} Invalid requests (${errorPercentage.toFixed(
                  1,
                )}%)`}
              >
                <span className="mr-[6px] flex-shrink-0">
                  <Ban iconSize="sm-regular" />
                </span>
                <span className="overflow-hidden text-ellipsis whitespace-nowrap max-w-[45px]">
                  {formatNumber(row.original.error_count)}
                </span>
              </Badge>
            </div>
            <div className="ml-2 flex-shrink-0">
              <OutcomesPopover
                outcomeCounts={row.original.outcome_counts}
                isSelected={selectedLog?.key_id === row.original.key_id}
              />
            </div>
          </div>
        );
      },
    },
    {
      id: "lastUsed",
      header: "Last Used",
      size: 150,
      enableSorting: true,
      cell: ({ row }) => (
        <TimestampInfo
          value={row.original.time}
          className={cn(
            "font-mono group-hover:underline decoration-dotted",
            selectedLog && selectedLog.request_id !== row.original.request_id && "pointer-events-none",
          )}
        />
      ),
    },
  ];

  return (
    <Table
      data={historicalLogs}
      columns={columns}
      isLoading={isLoading}
      isLoadingMore={isLoadingMore}
      onFetchMore={loadMore}
      hasMore={true}
      enableVirtualization={true}
      enableInfiniteScroll={true}
      enableSorting={true}
      manualSorting={false}
      enableGlobalFilter={false}
      enablePagination={false}
      enableRowSelection={false}
      enableColumnVisibility={false}
      enableExport={false}
      maxHeight="600px"
      getRowId={(log) => log.request_id}
      onRowClick={(log) => setSelectedLog(log)}
      rowClassName={(log) => getRowClassName(log, selectedLog as KeysOverviewLog)}
      emptyState={
        <div className="w-full flex justify-center items-center h-full">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>Key Verification Logs</Empty.Title>
            <Empty.Description className="text-left">
              No key verification data to show. Once requests are made with API keys, you'll see a
              summary of successful and failed verification attempts.
            </Empty.Description>{" "}
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
  );
};
