"use client";

import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { Ban, BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { OutcomesPopover } from "./components/outcome-popover";
import { KeyIdentifierColumn } from "./components/override-indicator";
import { useKeysOverviewLogsQuery } from "./hooks/use-logs-query";
import { STATUS_STYLES, getRowClassName, getStatusStyle } from "./utils/get-row-class";

const compactFormatter = new Intl.NumberFormat("en-US", {
  notation: "compact",
  maximumFractionDigits: 1,
});

export const KeysOverviewLogsTable = ({ apiId }: { apiId: string }) => {
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
                  STATUS_STYLES.success.badge.default,
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
                  style.badge.default,
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
        render: (log) => <OutcomesPopover outcomeCounts={log.outcome_counts} displayLimit={1} />,
      },
      {
        key: "lastUsed",
        header: "Last Used",
        width: "15%",
        render: (log) => (
          <div className="flex items-center gap-14 truncate text-accent-9">
            <TimestampInfo
              value={log.time}
              className={cn("font-mono group-hover:underline decoration-dotted")}
            />
          </div>
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
      keyExtractor={(log) => log.request_id}
      rowClassName={getRowClassName}
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
