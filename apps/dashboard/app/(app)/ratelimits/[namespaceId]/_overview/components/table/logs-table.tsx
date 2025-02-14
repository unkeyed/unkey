"use client";

import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";
import { ArrowDotAntiClockwise, Ban, BookBookmark, Focus } from "@unkey/icons";
import { Button, Empty, Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import ms from "ms";
import { compactFormatter } from "../../utils";
import { LogsTableAction } from "./components/logs-actions";
import { WarningWithTooltip } from "./components/warning-with-tooltip";
import { useRatelimitOverviewLogsQuery } from "./hooks/use-logs-query";
import { calculateBlockedPercentage } from "./utils/calculate-blocked-percentage";
import { STATUS_STYLES, getRowClassName, getStatusStyle } from "./utils/get-row-class";

const MAX_LATENCY = 10;
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
  const { historicalLogs, isLoading, isLoadingMore, loadMore } = useRatelimitOverviewLogsQuery({
    namespaceId,
  });

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
              Keep track of all activity within your workspace. Logs automatically record key
              actions like key creation, permission updates, and configuration changes, giving you a
              clear history of resource requests.
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

const columns = (namespaceId: string): Column<RatelimitOverviewLog>[] => {
  return [
    {
      key: "identifier",
      header: "Identifier",
      width: "7.5%",
      headerClassName: "pl-11",
      render: (log) => {
        const style = getStatusStyle(log);

        return (
          <div className="flex gap-6 items-center pl-2">
            <WarningWithTooltip log={log} />
            <div className="flex gap-3 items-center">
              <div className={cn(style.badge.default, "rounded p-1")}>
                {log.override ? (
                  <ArrowDotAntiClockwise size="md-regular" />
                ) : (
                  <Focus size="md-regular" />
                )}
              </div>
              <div className="font-mono text-accent-12 font-medium">{log.identifier}</div>
              {log.override && (
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <div className="group relative p-3 cursor-pointer -ml-[5px]">
                        <div className="absolute inset-0" />
                        <div
                          className={cn(
                            "size-[6px] rounded-full absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2",
                            calculateBlockedPercentage(log)
                              ? "bg-orange-10 hover:bg-orange-11"
                              : "bg-warning-10",
                          )}
                        />
                      </div>
                    </TooltipTrigger>
                    <TooltipContent
                      className="bg-gray-1 px-4 py-2 border border-accent-6 shadow-md font-medium text-xs text-accent-12 "
                      side="right"
                    >
                      <div className="flex gap-3 items-center">
                        <div
                          className={cn(
                            style.badge.default,
                            "rounded p-1",
                            "bg-accent-4 text-accent-11 group-hover:bg-accent-5",
                          )}
                        >
                          <ArrowDotAntiClockwise size="md-regular" />
                        </div>
                        <div className="flex flex-col gap-1">
                          <div className="text-sm flex gap-[10px] items-center">
                            <span className="font-medium text-sm text-accent-12">
                              Custom override in effect
                            </span>
                            <div className={cn("size-[6px] rounded-full bg-warning-10")} />
                          </div>
                          <div className="text-accent-9">
                            {log.override && (
                              <>
                                Limit set to{" "}
                                <span className="text-accent-12">
                                  {Intl.NumberFormat().format(log.override.limit)}{" "}
                                </span>
                                requests per{" "}
                                <span className="text-accent-12">{ms(log.override.duration)}</span>
                              </>
                            )}
                          </div>{" "}
                        </div>
                      </div>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              )}
            </div>
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
          <Badge
            className={cn(
              "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
              STATUS_STYLES.success.badge.default,
            )}
            title={`${log.passed_count.toLocaleString()} Passed requests`}
          >
            {compactFormatter.format(log.passed_count)}
          </Badge>
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
        );
      },
    },
    {
      key: "avgLatency",
      header: "Avg. Latency",
      width: "7.5%",
      render: (log) => (
        <div
          className={cn(
            "font-mono pr-4",
            log.avg_latency > MAX_LATENCY
              ? "text-orange-11 font-medium dark:text-warning-11"
              : "text-accent-9",
          )}
        >
          {Math.round(log.avg_latency)}ms
        </div>
      ),
    },
    {
      key: "p99",
      header: "P99 Latency",
      width: "7.5%",
      render: (log) => (
        <div
          className={cn(
            "font-mono pr-4",
            log.p99_latency > MAX_LATENCY
              ? "text-orange-11 font-medium dark:text-warning-11"
              : "text-accent-9",
          )}
        >
          {Math.round(log.p99_latency)}ms
        </div>
      ),
    },
    {
      key: "lastRequest",
      header: "Last Request",
      width: "7.5%",
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
