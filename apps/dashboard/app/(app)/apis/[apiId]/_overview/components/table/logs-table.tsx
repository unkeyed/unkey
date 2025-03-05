"use client";

import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { Ban, BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useKeysOverviewLogsQuery } from "./hooks/use-logs-query";
import {
  getRowClassName,
  getStatusStyle,
  STATUS_STYLES,
} from "./utils/get-row-class";
import { KeyIdentifierColumn } from "./components/override-indicator";

const compactFormatter = new Intl.NumberFormat("en-US", {
  notation: "compact",
  maximumFractionDigits: 1,
});

export const KeysOverviewLogsTable = ({ apiId }: { apiId: string }) => {
  const { historicalLogs, isLoading, isLoadingMore, loadMore } =
    useKeysOverviewLogsQuery({
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
          return (
            <div className="flex gap-3 items-center group/name truncate font-mono text-accent-12">
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
            <div className="flex gap-3 items-center">
              <Badge
                className={cn(
                  "px-[6px] rounded-md font-mono whitespace-nowrap",
                  STATUS_STYLES.success.badge.default
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
          return (
            <div className="flex gap-3 items-center">
              <Badge
                className={cn(
                  "px-[6px] rounded-md font-mono whitespace-nowrap gap-[6px]",
                  style.badge.default
                )}
                title={`${log.error_count.toLocaleString()} Invalid requests`}
              >
                {log.error_count > 0 && <Ban size="sm-regular" />}
                {compactFormatter.format(log.error_count)}
              </Badge>
            </div>
          );
        },
      },
      {
        key: "outcomes",
        header: "Other Outcomes",
        width: "25%",
        render: (log) => {
          // Filter out outcomes to show (showing only non-zero counts)
          const relevantOutcomes = Object.entries(log.outcome_counts).filter(
            ([outcome, count]) => count > 0 && outcome !== "VALID"
          );

          return (
            <div className="flex flex-wrap gap-1">
              {relevantOutcomes.map(([outcome, count]) => (
                <Badge
                  key={outcome}
                  className={cn(
                    "px-[6px] rounded-md font-mono whitespace-nowrap",
                    getOutcomeBadgeClass(outcome)
                  )}
                  title={`${count.toLocaleString()} ${formatOutcomeName(
                    outcome
                  )} requests`}
                >
                  {formatOutcomeName(outcome)}: {compactFormatter.format(count)}
                </Badge>
              ))}
            </div>
          );
        },
      },
      {
        key: "lastUsed",
        header: "Last Used",
        width: "15%",
        render: (log) => (
          <div className="flex items-center gap-14 truncate text-accent-9">
            <TimestampInfo
              value={log.time}
              className={cn(
                "font-mono group-hover:underline decoration-dotted"
              )}
            />
          </div>
        ),
      },
    ];
  };

  // Helper function to format outcome names
  const formatOutcomeName = (outcome: string): string => {
    if (!outcome) {
      return "Unknown";
    }

    return outcome
      .split("_")
      .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
      .join(" ");
  };

  // Helper function to get the appropriate badge class for an outcome
  const getOutcomeBadgeClass = (outcome: string): string => {
    switch (outcome) {
      case "VALID":
        return "bg-accent-4 text-accent-11";
      case "RATE_LIMITED":
        return "bg-warning-4 text-warning-11";
      case "INSUFFICIENT_PERMISSIONS":
      case "FORBIDDEN":
        return "bg-error-4 text-error-11";
      case "DISABLED":
      case "EXPIRED":
      case "USAGE_EXCEEDED":
        return "bg-gray-4 text-gray-11";
      default:
        return "bg-blue-4 text-blue-11";
    }
  };

  return (
    <VirtualTable
      data={historicalLogs}
      isLoading={isLoading}
      isFetchingNextPage={isLoadingMore}
      onLoadMore={loadMore}
      columns={columns()}
      keyExtractor={(log) => log.request_id}
      rowClassName={getRowClassName}
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
