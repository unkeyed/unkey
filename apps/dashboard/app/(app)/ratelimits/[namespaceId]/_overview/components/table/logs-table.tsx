"use client";

import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import { Ban, BookBookmark, Focus, TriangleWarning2 } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useMemo } from "react";
import { type RatelimitOverviewLogs, generateMockApiData } from "../../dev-utils";
import { compactFormatter } from "../../utils";

const MAX_LATENCY = 10;

type StatusStyle = {
  base: string;
  hover: string;
  selected: string;
  badge: {
    default: string;
    selected: string;
  };
  focusRing: string;
};

const STATUS_STYLES = {
  success: {
    base: "text-accent-9",
    hover: "hover:bg-accent-3",
    selected: "text-accent-11 bg-accent-3",
    badge: {
      default: "bg-accent-4 text-accent-11 group-hover:bg-accent-5",
      selected: "bg-accent-5 text-accent-12 hover:bg-hover-5",
    },
    focusRing: "focus:ring-accent-7",
  },
  blocked: {
    base: "text-orange-11",
    hover: "hover:bg-orange-3",
    selected: "bg-orange-3",
    badge: {
      default: "bg-orange-4 text-orange-11 group-hover:bg-orange-5",
      selected: "bg-orange-5 text-orange-11 hover:bg-orange-5",
    },
    focusRing: "focus:ring-orange-7",
  },
};

const getStatusStyle = (passed: number, blocked: number): StatusStyle => {
  return blocked > passed ? STATUS_STYLES.blocked : STATUS_STYLES.success;
};

const historicalLogs = generateMockApiData();
export const RatelimitOverviewLogsTable = () => {
  const getRowClassName = (log: RatelimitOverviewLogs) => {
    const hasMoreBlocked = log.blocked > log.passed;
    const style = getStatusStyle(log.passed, log.blocked);

    return cn(
      style.base,
      style.hover,
      hasMoreBlocked ? "bg-orange-2" : "",
      "group rounded-md",
      "focus:outline-none focus:ring-1 focus:ring-opacity-40",
      style.focusRing,
    );
  };

  const columns: Column<RatelimitOverviewLogs>[] = useMemo(
    () => [
      {
        key: "identifier",
        header: "Identifier",
        width: "7.5%",
        headerClassName: "pl-11",
        render: (log) => {
          const style = getStatusStyle(log.passed, log.blocked);

          return (
            <div className="flex gap-6 items-center pl-2">
              <div className={cn(log.blocked > log.passed ? "block" : "invisible")}>
                <TriangleWarning2 />
              </div>
              <div className="flex gap-3 items-center">
                <div className={cn(style.badge.default, "rounded p-1")}>
                  <Focus size="md-regular" />
                </div>
                <div className="font-mono pr-4 text-accent-12 font-medium">{log.identifier}</div>
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
              title={`${log.passed.toLocaleString()} Passed requests`}
            >
              {compactFormatter.format(log.passed)}
            </Badge>
          );
        },
      },
      {
        key: "blocked",
        header: "Blocked",
        width: "7.5%",
        render: (log) => {
          const style = getStatusStyle(log.passed, log.blocked);
          return (
            <Badge
              className={cn(
                "uppercase px-[6px] rounded-md font-mono whitespace-nowrap gap-[6px]",
                style.badge.default,
              )}
              title={`${log.blocked.toLocaleString()} Blocked requests`}
            >
              <Ban size="sm-regular" />
              {compactFormatter.format(log.blocked)}
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
              log.avgLatency > MAX_LATENCY
                ? "text-orange-11 font-medium dark:text-warning-11"
                : "text-accent-9",
            )}
          >
            {log.avgLatency.toFixed(2)}ms
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
              log.p99Latency > MAX_LATENCY
                ? "text-orange-11 font-medium dark:text-warning-11"
                : "text-accent-9",
            )}
          >
            {log.p99Latency}ms
          </div>
        ),
      },
      {
        key: "lastRequest",
        header: "Last Request",
        width: "7.5%",
        render: (log) => (
          <div className="flex items-center gap-3 truncate text-accent-9">
            <TimestampInfo
              value={log.lastRequest}
              className={cn("font-mono group-hover:underline decoration-dotted")}
            />
          </div>
        ),
      },
    ],
    [],
  );

  return (
    <VirtualTable
      data={historicalLogs}
      columns={columns}
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
