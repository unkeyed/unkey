"use client";

import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import { Badge, Empty, TimestampInfo } from "@unkey/ui";
import { useMemo } from "react";
import { useRuntimeLogs } from "../../context/runtime-logs-provider";
import { useRuntimeLogsFilters } from "../../hooks/use-runtime-logs-filters";
import type { RuntimeLog } from "../../types";
import { useRuntimeLogsQuery } from "./hooks/use-runtime-logs-query";

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

const STATUS_STYLES: Record<"success" | "warning" | "error", StatusStyle> = {
  success: {
    base: "text-grayA-9",
    hover: "hover:text-accent-11 dark:hover:text-accent-12 hover:bg-grayA-3",
    selected: "text-accent-12 bg-grayA-3 hover:text-accent-12",
    badge: {
      default: "bg-grayA-3 text-grayA-11 group-hover:bg-grayA-5",
      selected: "bg-grayA-5 text-grayA-12 hover:bg-grayA-5",
    },
    focusRing: "focus:ring-accent-7",
  },
  warning: {
    base: "text-warning-11 bg-warning-2",
    hover: "hover:bg-warning-3",
    selected: "bg-warning-3",
    badge: {
      default: "bg-warning-4 text-warning-11 group-hover:bg-warning-5",
      selected: "bg-warning-5 text-warning-11 hover:bg-warning-5",
    },
    focusRing: "focus:ring-warning-7",
  },
  error: {
    base: "text-error-11 bg-error-2",
    hover: "hover:bg-error-3",
    selected: "bg-error-3",
    badge: {
      default: "bg-error-4 text-error-11 group-hover:bg-error-5",
      selected: "bg-error-5 text-error-11 hover:bg-error-5",
    },
    focusRing: "focus:ring-error-7",
  },
};

const getSeverityStyle = (severity: string): StatusStyle => {
  const upper = severity.toUpperCase();
  if (upper === "ERROR") {
    return STATUS_STYLES.error;
  }
  if (upper === "WARN" || upper === "WARNING") {
    return STATUS_STYLES.warning;
  }
  return STATUS_STYLES.success;
};

const getLogKey = (log: RuntimeLog): string => `${log.time}-${log.region}-${log.message}`;

const getSelectedClassName = (log: RuntimeLog, isSelected: boolean): string => {
  if (!isSelected) {
    return "";
  }
  return getSeverityStyle(log.severity).selected;
};

export function RuntimeLogsTable() {
  const { filters } = useRuntimeLogsFilters();
  const { selectedLog, setSelectedLog } = useRuntimeLogs();
  const { logs, total, isLoading, hasMore, loadMore, isLoadingMore } = useRuntimeLogsQuery({
    filters,
  });

  const selectedLogKey = selectedLog ? getLogKey(selectedLog) : null;

  const columns: Column<RuntimeLog>[] = useMemo(
    () => [
      {
        key: "time",
        header: "Time",
        width: "12%",
        headerClassName: "pl-4",
        render: (log) => (
          <div className="pl-4">
            <TimestampInfo
              value={log.time}
              className={cn(
                "font-mono group-hover:underline decoration-dotted",
                selectedLogKey && selectedLogKey !== getLogKey(log) && "pointer-events-none",
              )}
            />
          </div>
        ),
      },
      {
        key: "severity",
        header: "Severity",
        width: "10%",
        render: (log) => {
          const style = getSeverityStyle(log.severity);
          const isSelected = selectedLogKey === getLogKey(log);
          return (
            <Badge
              className={cn(
                "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
                isSelected ? style.badge.selected : style.badge.default,
              )}
            >
              {log.severity}
            </Badge>
          );
        },
      },
      {
        key: "region",
        header: "Region",
        width: "10%",
        render: (log) => (
          <div className="font-mono pr-4 truncate uppercase" title={log.region}>
            {log.region}
          </div>
        ),
      },
      {
        key: "message",
        header: "Message",
        width: "20%",
        render: (log) => (
          <div className="font-mono truncate pr-4 max-w-[300px]" title={log.message}>
            {log.message}
          </div>
        ),
      },
      {
        key: "attributes",
        header: "Attributes",
        width: "auto",
        render: (log) => {
          let attrStr = log.attributes ? JSON.stringify(log.attributes) : "";
          if (attrStr === "" || attrStr === "{}") {
            attrStr = "—";
          }
          return (
            <div className="font-mono truncate text-xs max-w-[500px]" title={attrStr}>
              {attrStr}
            </div>
          );
        },
      },
    ],
    [selectedLogKey],
  );

  const getRowClassName = (log: RuntimeLog): string => {
    const style = getSeverityStyle(log.severity);
    const isSelected = selectedLogKey === getLogKey(log);

    return cn(
      style.base,
      style.hover,
      "group rounded-md",
      "focus:outline-hidden focus:ring-1 focus:ring-opacity-40",
      style.focusRing,
      isSelected && style.selected,
      selectedLogKey && {
        "opacity-50 z-0": !isSelected,
        "opacity-100 z-10": isSelected,
      },
    );
  };

  return (
    <VirtualTable
      data={logs}
      isLoading={isLoading}
      isFetchingNextPage={isLoadingMore}
      onLoadMore={loadMore}
      columns={columns}
      onRowClick={setSelectedLog}
      selectedItem={selectedLog}
      keyExtractor={getLogKey}
      rowClassName={getRowClassName}
      selectedClassName={getSelectedClassName}
      loadMoreFooterProps={{
        hide: isLoading,
        buttonText: "Load more logs",
        hasMore,
        countInfoText: (
          <div className="flex gap-2">
            <span>Showing</span>
            <span className="text-accent-12">{new Intl.NumberFormat().format(logs.length)}</span>
            <span>of</span>
            <span className="text-accent-12">{new Intl.NumberFormat().format(total)}</span>
            <span>logs</span>
          </div>
        ),
      }}
      emptyState={
        <div className="w-full flex justify-center items-center h-full">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>No runtime logs</Empty.Title>
            <Empty.Description className="text-left">
              No logs found for the selected filters and time range.
            </Empty.Description>
          </Empty>
        </div>
      }
    />
  );
}
