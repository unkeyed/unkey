"use client";

import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import type { Log } from "@unkey/clickhouse/src/logs";
import { TriangleWarning2 } from "@unkey/icons";
import { generateMockLogs } from "./utils";

const logs = generateMockLogs(50);

type Props = {
  onLogSelect: (log: Log | null) => void;
  selectedLog: Log | null;
};

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

const createStatusStyle = (colors: {
  base: string;
  hover: string;
  selected: string;
  badge: { bg: string; text: string; hoverBg: string; selectedBg: string };
  ring: string;
}): StatusStyle => ({
  base: `${colors.base} text-${colors.badge.text}`,
  hover: colors.hover,
  selected: colors.selected,
  badge: {
    default: `bg-${colors.badge.bg} text-${colors.badge.text} group-hover:bg-${colors.badge.hoverBg}`,
    selected: `bg-${colors.badge.selectedBg} text-${colors.badge.text}`,
  },
  focusRing: `ring-${colors.ring}`,
});

const STATUS_STYLES = {
  success: createStatusStyle({
    base: "text-accent-9",
    hover: "hover:text-accent-11 dark:hover:text-accent-12",
    selected: "text-accent-11 bg-accent-3 dark:text-accent-12",
    badge: {
      bg: "accent-4",
      text: "accent-11",
      hoverBg: "accent-5",
      selectedBg: "accent-5",
    },
    ring: "accent-7",
  }),
  warning: createStatusStyle({
    base: "bg-warning-2",
    hover: "hover:bg-warning-3",
    selected: "bg-warning-3",
    badge: {
      bg: "warning-4",
      text: "warning-11",
      hoverBg: "warning-5",
      selectedBg: "warning-5",
    },
    ring: "warning-7",
  }),
  error: createStatusStyle({
    base: "bg-error-2",
    hover: "hover:bg-error-3",
    selected: "bg-error-3",
    badge: {
      bg: "error-4",
      text: "error-11",
      hoverBg: "error-5",
      selectedBg: "error-5",
    },
    ring: "error-7",
  }),
};

const METHOD_BADGE = {
  base: "uppercase px-[6px] rounded-md font-mono bg-accent-4 text-accent-11 group-hover:bg-accent-5 group-hover:text-accent-12",
  selected: "bg-accent-5 text-accent-12",
};

const getStatusStyle = (status: number): StatusStyle => {
  if (status >= 500) {
    return STATUS_STYLES.error;
  }
  if (status >= 400) {
    return STATUS_STYLES.warning;
  }
  return STATUS_STYLES.success;
};

const WARNING_ICON_STYLES = {
  base: "size-3",
  warning: "text-warning-11",
  error: "text-error-11",
};

const getSelectedClassName = (log: Log, isSelected: boolean) => {
  if (!isSelected) {
    return "";
  }
  return getStatusStyle(log.response_status).selected;
};

export const LogsTable = ({ onLogSelect, selectedLog }: Props) => {
  const getRowClassName = (log: Log) => {
    const style = getStatusStyle(log.response_status);
    const isSelected = selectedLog?.request_id === log.request_id;

    return cn(
      style.base,
      style.hover,
      "group",
      "focus:outline-none focus:ring-1 focus:ring-opacity-40",
      style.focusRing,
      // This creates a spotlight effect
      selectedLog && {
        "opacity-50 z-0": !isSelected,
        "opacity-100 z-10": isSelected,
      },
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
              WARNING_ICON_STYLES.base,
              log.response_status < 300 && "invisible",
              log.response_status >= 400 &&
                log.response_status < 500 &&
                WARNING_ICON_STYLES.warning,
              log.response_status >= 500 && WARNING_ICON_STYLES.error,
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
      render: (log) => {
        const style = getStatusStyle(log.response_status);
        const isSelected = selectedLog?.request_id === log.request_id;
        return (
          <Badge
            className={cn(
              "uppercase px-[6px] rounded-md font-mono",
              isSelected ? style.badge.selected : style.badge.default,
            )}
          >
            {log.response_status}
          </Badge>
        );
      },
    },
    {
      key: "method",
      header: "Method",
      width: "78px",
      render: (log) => {
        const isSelected = selectedLog?.request_id === log.request_id;
        return (
          <Badge className={cn(METHOD_BADGE.base, isSelected && METHOD_BADGE.selected)}>
            {log.method}
          </Badge>
        );
      },
    },
    {
      key: "path",
      header: "Path",
      width: "15%",
      render: (log) => <div className="flex items-center gap-2 font-mono truncate">{log.path}</div>,
    },
    {
      key: "response",
      header: "Response Body",
      width: "1fr",
      render: (log) => <span className="truncate font-mono">{log.response_body}</span>,
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
      selectedClassName={getSelectedClassName}
    />
  );
};
