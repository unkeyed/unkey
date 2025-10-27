"use client";

import { useSort } from "@/components/logs/hooks/use-sort";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { formatNumber } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";
import { Ban, BookBookmark } from "@unkey/icons";
import { Badge, Button, Empty, TimestampInfo } from "@unkey/ui";
import { useState } from "react";
import { InlineFilter } from "./components/inline-filter";
import { LogsTableAction } from "./components/logs-actions";
import { IdentifierColumn } from "./components/override-indicator";
import { useRatelimitOverviewLogsQuery } from "./hooks/use-logs-query";
import type { SortFields } from "./query-logs.schema";
import { STATUS_STYLES, getRowClassName, getStatusStyle } from "./utils/get-row-class";

// const MAX_LATENCY = 10;
export const RatelimitOverviewLogsTable = ({
  namespaceId,
}: {
  namespaceId: string;
}) => {
  const [selectedLog, setSelectedLog] = useState<RatelimitOverviewLog>();
  const { getSortDirection, toggleSort } = useSort<SortFields>();
  const { historicalLogs, isLoading, isLoadingMore, loadMore, hasMore, totalCount } =
    useRatelimitOverviewLogsQuery({
      namespaceId,
    });

  const columns = (namespaceId: string): Column<RatelimitOverviewLog>[] => {
    return [
      {
        key: "identifier",
        header: "Identifier",
        width: "20%",
        headerClassName: "pl-12",
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
        width: "20%",
        sort: {
          direction: getSortDirection("passed"),
          sortable: true,
          onSort() {
            toggleSort("passed", false);
          },
        },
        render: (log) => {
          return (
            <div className="flex gap-3 items-center group/identifier">
              <Badge
                className={cn(
                  "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
                  selectedLog?.request_id === log.request_id
                    ? STATUS_STYLES.success.badge.selected
                    : STATUS_STYLES.success.badge.default,
                )}
                title={`${log.passed_count.toLocaleString()} Passed requests`}
              >
                {formatNumber(log.passed_count)}
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
        width: "20%",
        sort: {
          direction: getSortDirection("blocked"),
          sortable: true,
          onSort() {
            toggleSort("blocked", false);
          },
        },
        render: (log) => {
          const style = getStatusStyle(log);
          return (
            <div className="flex gap-3 items-center group/identifier">
              <Badge
                className={cn(
                  "uppercase px-[6px] rounded-md font-mono whitespace-nowrap gap-[6px]",
                  selectedLog?.request_id === log.request_id
                    ? style.badge.selected
                    : style.badge.default,
                )}
                title={`${log.blocked_count.toLocaleString()} Blocked requests`}
              >
                <Ban iconSize="sm-regular" />
                {formatNumber(log.blocked_count)}
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
        width: "20%",
        sort: {
          direction: getSortDirection("time"),
          sortable: true,
          onSort() {
            toggleSort("time", false, "asc");
          },
        },
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
      {
        key: "actions",
        header: "",
        width: "auto",
        render: (log) => (
          <LogsTableAction
            overrideDetails={log.override}
            identifier={log.identifier}
            namespaceId={namespaceId}
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
      onRowClick={setSelectedLog}
      selectedItem={selectedLog}
      onLoadMore={loadMore}
      columns={columns(namespaceId)}
      keyExtractor={(log) => log.identifier}
      rowClassName={(rowLog) => getRowClassName(rowLog, selectedLog as RatelimitOverviewLog)}
      loadMoreFooterProps={{
        itemLabel: "identifiers",
        buttonText: "Load more logs",
        hasMore,
        hide: isLoading,
        countInfoText: (
          <div className="flex gap-2">
            <span>Showing</span> <span className="text-accent-12">{historicalLogs.length}</span>
            <span>of {totalCount}</span>
            <span>rate limit identifiers</span>
          </div>
        ),
      }}
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
