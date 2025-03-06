"use client";
import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { Ban, BookBookmark, TriangleWarning } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useState } from "react";

import { OutcomesPopover } from "./components/outcome-popover";
import { KeyIdentifierColumn } from "./components/override-indicator";
import { useKeysOverviewLogsQuery } from "./hooks/use-logs-query";
import {
  getErrorSeverity,
  getErrorPercentage,
} from "./utils/calculate-blocked-percentage";
import {
  STATUS_STYLES,
  getStatusStyle,
  getRowClassName,
} from "./utils/get-row-class";

const compactFormatter = new Intl.NumberFormat("en-US", {
  notation: "compact",
  maximumFractionDigits: 1,
});

export const KeysOverviewLogsTable = ({ apiId }: { apiId: string }) => {
  const [selectedLog, setSelectedLog] = useState<KeysOverviewLog>();
  const { historicalLogs, isLoading, isLoadingMore, loadMore } =
    useKeysOverviewLogsQuery({
      apiId,
    });

  // Helper to determine icon based on severity
  const getStatusIcon = (log: KeysOverviewLog) => {
    const severity = getErrorSeverity(log);

    switch (severity) {
      case "high":
        return <Ban size="sm-regular" className="text-error-11" />;
      case "moderate":
        return <Ban size="sm-regular" className="text-orange-11" />;
      case "low":
        return <Ban size="sm-regular" className="text-warning-11" />;
      default:
        return <Ban size="sm-regular" className="text-accent-11" />;
    }
  };

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
          return (
            <div className="flex gap-3 items-center group/name truncate font-mono">
              {log.key_details?.name || "<EMPTY>"}
            </div>
          );
        },
      },
      {
        key: "valid",
        header: "Valid",
        width: "15%",
        render: (log) => {
          return (
            <div className="flex gap-3 items-center tabular-nums">
              <Badge
                className={cn(
                  "px-[6px] rounded-md font-mono whitespace-nowrap",
                  selectedLog?.key_id === log.key_id
                    ? STATUS_STYLES.success.badge.selected
                    : STATUS_STYLES.success.badge.default
                )}
                title={`${log.valid_count.toLocaleString()} Valid requests`}
              >
                {compactFormatter.format(log.valid_count)}
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
              <div className="flex-shrink-0 w-[50px]">
                <Badge
                  className={cn(
                    "px-[6px] rounded-md font-mono whitespace-nowrap flex items-center",
                    selectedLog?.key_id === log.key_id
                      ? style.badge.selected
                      : style.badge.default
                  )}
                  title={`${log.error_count.toLocaleString()} Invalid requests (${errorPercentage.toFixed(
                    1
                  )}%)`}
                >
                  <span className="mr-[6px] flex-shrink-0">
                    {getStatusIcon(log)}
                  </span>
                  <span>{compactFormatter.format(log.error_count)}</span>
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
              selectedLog &&
                selectedLog.request_id !== log.request_id &&
                "pointer-events-none"
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
      rowClassName={(log) =>
        getRowClassName(log, selectedLog as KeysOverviewLog)
      }
      emptyState={
        <div className="w-full flex justify-center items-center h-full">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>Key Verification Logs</Empty.Title>
            <Empty.Description className="text-left">
              No key verification data to show. Once requests are made with API
              keys, you'll see a summary of successful and failed verification
              attempts.
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
