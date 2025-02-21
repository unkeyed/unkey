"use client";

import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";
import { Ban, BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { compactFormatter } from "../../utils";
import { InlineFilter } from "./components/inline-filter";
import { LogsTableAction } from "./components/logs-actions";
import { IdentifierColumn } from "./components/override-indicator";
import { useRatelimitOverviewLogsQuery } from "./hooks/use-logs-query";
import { useSort } from "./hooks/use-sort";
import { STATUS_STYLES, getRowClassName, getStatusStyle } from "./utils/get-row-class";

// const MAX_LATENCY = 10;
export type OverrideDetails = {
  overrideId?: string;
  limit: number;
  duration: number;
  async?: boolean | null;
};

export const RatelimitOverviewLogsTable = ({
  namespaceId,
}: {
  namespaceId: string;
}) => {
  const { getSortDirection, toggleSort } = useSort();
  const { historicalLogs, isLoading, isLoadingMore, loadMore } = useRatelimitOverviewLogsQuery({
    namespaceId,
  });

  const columns = (namespaceId: string): Column<RatelimitOverviewLog>[] => {
    return [
      {
        key: "identifier",
        header: "Identifier",
        width: "7.5%",
        headerClassName: "pl-11",
        render: (log) => {
          return (
            <div className="flex gap-3 items-center group/identifier">
              <IdentifierColumn log={log} />
              <InlineFilter
                content="Filter by identifier"
                filterPair={{ identifiers: log.identifier }}
              />
            </div>
          );
        },
      },
      {
        key: "passed",
        header: "Passed",
        width: "7.5%",
        render: (log) => {
          return (
            <div className="flex gap-3 items-center group/identifier">
              <Badge
                className={cn(
                  "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
                  STATUS_STYLES.success.badge.default,
                )}
                title={`${log.passed_count.toLocaleString()} Passed requests`}
              >
                {compactFormatter.format(log.passed_count)}
              </Badge>
              <InlineFilter
                filterPair={{ identifiers: log.identifier, status: "passed" }}
                content="Filter by identifier and passed status"
              />
            </div>
          );
        },
      },
      {
        key: "blocked",
        header: "Blocked",
        width: "7.5%",
        render: (log) => {
          const style = getStatusStyle(log);
          return (
            <div className="flex gap-3 items-center group/identifier">
              <Badge
                className={cn(
                  "uppercase px-[6px] rounded-md font-mono whitespace-nowrap gap-[6px]",
                  style.badge.default,
                )}
                title={`${log.blocked_count.toLocaleString()} Blocked requests`}
              >
                <Ban size="sm-regular" />
                {compactFormatter.format(log.blocked_count)}
              </Badge>
              <InlineFilter
                content="Filter by identifier and blocked status"
                filterPair={{ identifiers: log.identifier, status: "blocked" }}
              />
            </div>
          );
        },
      },
      // {
      //   key: "avgLatency",
      //   header: "Avg. Latency",
      //   width: "7.5%",
      //   sort: {
      //     direction: getSortDirection("avg_latency"),
      //     sortable: true,
      //     onSort() {
      //       toggleSort("avg_latency", true);
      //     },
      //   },
      //   render: (log) => (
      //     <div
      //       className={cn(
      //         "font-mono pr-4",
      //         log.avg_latency > MAX_LATENCY
      //           ? "text-orange-11 font-medium dark:text-warning-11"
      //           : "text-accent-9",
      //       )}
      //     >
      //       {Math.round(log.avg_latency)}ms
      //     </div>
      //   ),
      // },
      // {
      //   key: "p99",
      //   header: "P99 Latency",
      //   width: "7.5%",
      //   sort: {
      //     direction: getSortDirection("p99_latency"),
      //     sortable: true,
      //     onSort() {
      //       toggleSort("p99_latency", true);
      //     },
      //   },
      //   render: (log) => (
      //     <div
      //       className={cn(
      //         "font-mono pr-4",
      //         log.p99_latency > MAX_LATENCY
      //           ? "text-orange-11 font-medium dark:text-warning-11"
      //           : "text-accent-9",
      //       )}
      //     >
      //       {Math.round(log.p99_latency)}ms
      //     </div>
      //   ),
      // },
      {
        key: "lastRequest",
        header: "Last Request",
        width: "7.5%",
        sort: {
          direction: getSortDirection("time"),
          sortable: true,
          onSort() {
            toggleSort("time", true);
          },
        },
        render: (log) => (
          <div className="flex items-center gap-14 truncate text-accent-9">
            <TimestampInfo
              value={log.time}
              className={cn("font-mono group-hover:underline decoration-dotted")}
            />
            <div className="invisible group-hover:visible">
              <LogsTableAction
                overrideDetails={log.override}
                identifier={log.identifier}
                namespaceId={namespaceId}
              />
            </div>
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
      columns={columns(namespaceId)}
      keyExtractor={(log) => log.identifier}
      rowClassName={getRowClassName}
      emptyState={
        <div className="w-full flex justify-center items-center h-full">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>Logs</Empty.Title>
            <Empty.Description className="text-left">
              No rate limit data to show. Once requests are made, you'll see a summary of passed and
              blocked requests for each rate limit identifier.
            </Empty.Description>
            <Empty.Actions className="mt-4 justify-start">
              <a
                href="https://www.unkey.com/docs/introduction"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button>
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
