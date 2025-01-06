"use client";

import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { type Column, VirtualTable } from "@/components/virtual-table";
import { cn } from "@/lib/utils";
import type { Log } from "@unkey/clickhouse/src/logs";
import { TriangleWarning2 } from "@unkey/icons";
import { generateMockLogs } from "./utils";

const logs = generateMockLogs(50);

type Props = {
  onLogSelect: (log: Log | null) => void;
  selectedLog: Log | null;
};

const STATUS_STYLES = {
  success: {
    base: "text-accent-9",
    hover: "hover:text-accent-11 dark:hover:text-accent-12",
    selected: "text-accent-11 bg-accent-3 dark:text-accent-12",
    badge:
      "bg-accent-4 text-accent-11 group-hover:bg-accent-5 group-hover:text-accent-12",
    focusRing: "ring-accent-7",
    badgeSelected: "[&_.status]:bg-accent-5 [&_.status]:text-accent-12",
  },
  warning: {
    base: "bg-warning-2 text-warning-11",
    hover: "hover:bg-warning-3",
    selected: "bg-warning-3 text-warning-11",
    badge: "bg-warning-4 text-warning-11 group-hover:bg-warning-5",
    focusRing: "ring-warning-7",
    badgeSelected: "[&_.status]:bg-warning-5 [&_.status]:text-warning-12",
  },
  error: {
    base: "bg-error-2 text-error-11",
    hover: "hover:bg-error-3",
    selected: "bg-error-3 text-error-11",
    badge: "bg-error-4 text-error-11 group-hover:bg-error-5",
    focusRing: "ring-error-7",
    badgeSelected: "[&_.status]:bg-error-5 [&_.status]:text-error-12",
  },
};

const METHOD_STYLES = {
  base: "method uppercase px-[6px] rounded-md font-mono bg-accent-4 text-accent-11 group-hover:text-accent-12 group-hover:bg-accent-5",
  selected: "[&_.method]:bg-accent-5 [&_.method]:text-accent-12",
};

const getFocusRingStyles = (status: number) => {
  if (status >= 500) return "ring-error-7";
  if (status >= 400) return "ring-warning-7";
  return "ring-accent-7";
};

const getStatusStyle = (status: number) => {
  if (status >= 500) return STATUS_STYLES.error;
  if (status >= 400) return STATUS_STYLES.warning;
  return STATUS_STYLES.success;
};

export const LogsTable = ({ onLogSelect, selectedLog }: Props) => {
  const getRowClassName = (log: Log) => {
    const style = getStatusStyle(log.response_status);
    return cn(style.base, style.hover);
  };

  const getSelectedClassName = (log: Log, isSelected: boolean) => {
    if (!isSelected) return "";
    const style = getStatusStyle(log.response_status);
    return cn(style.selected, style.badgeSelected, METHOD_STYLES.selected);
  };

  const getFocusClassName = (log: Log) => {
    const ringStyle = getFocusRingStyles(log.response_status);
    return cn(
      "focus:outline-none focus:ring-1 focus:ring-opacity-40",
      ringStyle
    );
  };

  const columns: Column<Log>[] = [
    {
      key: "time",
      header: "Time",
      width: "165px",
      headerClassName: "pl-9",
      render: (log) => (
        <div className="flex items-center gap-3 px-2">
          <TriangleWarning2
            className={cn(
              "size-3",
              log.response_status < 300 && "invisible",
              log.response_status >= 400 &&
                log.response_status < 500 &&
                "text-warning-11",
              log.response_status >= 500 && "text-error-11"
            )}
          />
          <TimestampInfo value={log.time} />
        </div>
      ),
    },
    {
      key: "status",
      header: "Status",
      width: "78px",
      render: (log) => (
        <Badge
          className={cn(
            "status uppercase px-[6px] rounded-md font-mono",
            getStatusStyle(log.response_status).badge
          )}
        >
          {log.response_status}
        </Badge>
      ),
    },
    {
      key: "method",
      header: "Method",
      width: "78px",
      render: (log) => (
        <Badge className={METHOD_STYLES.base}>{log.method}</Badge>
      ),
    },
    {
      key: "path",
      header: "Path",
      width: "15%",
      render: (log) => (
        <div className="flex items-center gap-2 font-mono truncate">
          {log.path}
        </div>
      ),
    },
    {
      key: "response",
      header: "Response Body",
      width: "1fr",
      render: (log) => (
        <span className="truncate font-mono">{log.response_body}</span>
      ),
    },
  ];

  return (
    <VirtualTable
      data={logs ?? []}
      columns={columns}
      onRowClick={onLogSelect}
      selectedItem={selectedLog}
      keyExtractor={(log) => log.request_id}
      rowClassName={getRowClassName}
      focusClassName={getFocusClassName}
      selectedClassName={getSelectedClassName}
    />
  );
};
