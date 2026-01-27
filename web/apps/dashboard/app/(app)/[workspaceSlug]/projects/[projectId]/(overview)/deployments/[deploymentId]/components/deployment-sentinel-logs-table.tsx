"use client";

import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import type { SentinelResponse } from "@unkey/clickhouse/src/sentinel";
import { BookBookmark, TriangleWarning2 } from "@unkey/icons";
import { Badge, Button, Empty, TimestampInfo } from "@unkey/ui";
import { useState } from "react";
import {
  WARNING_ICON_STYLES,
  getStatusStyle,
} from "../../../sentinel-logs/components/table/utils/get-row-class";
import { useDeploymentSentinelLogsQuery } from "../hooks/use-deployment-sentinel-logs-query";

type ResponseBody = {
  keyId: string;
  valid: boolean;
  meta: Record<string, unknown>;
  enabled: boolean;
  permissions: string[];
  code:
  | "VALID"
  | "RATE_LIMITED"
  | "EXPIRED"
  | "USAGE_EXCEEDED"
  | "DISABLED"
  | "FORBIDDEN"
  | "INSUFFICIENT_PERMISSIONS";
};

const extractResponseField = <K extends keyof ResponseBody>(
  log: SentinelResponse,
  fieldName: K,
): ResponseBody[K] | null => {
  if (!log?.response_body) {
    return null;
  }

  try {
    const parsedBody = JSON.parse(log.response_body) as ResponseBody;
    return parsedBody[fieldName];
  } catch {
    return null;
  }
};

const getSelectedClassName = (log: SentinelResponse, isSelected: boolean) => {
  if (!isSelected) {
    return "";
  }
  const style = getStatusStyle(log.response_status);
  return style.selected;
};

const getRowClassName = (log: SentinelResponse, selectedLog: SentinelResponse | null): string => {
  if (!log?.request_id) {
    throw new Error("Log must have a valid request_id");
  }

  if (
    !Number.isInteger(log.response_status) ||
    log.response_status < 100 ||
    log.response_status > 599
  ) {
    throw new Error(
      `Invalid response_status: ${log.response_status}. Must be a valid HTTP status code.`,
    );
  }

  const style = getStatusStyle(log.response_status);
  const isSelected = Boolean(selectedLog?.request_id === log.request_id);

  const baseClasses = [
    style.base,
    style.hover,
    "group rounded-md",
    "focus:outline-none focus:ring-1 focus:ring-opacity-40",
    style.focusRing,
  ];

  const conditionalClasses = [
    isSelected && style.selected,
    selectedLog && {
      "opacity-50 z-0": !isSelected,
      "opacity-100 z-10": isSelected,
    },
  ].filter(Boolean);

  return cn(...baseClasses, ...conditionalClasses);
};

export const DeploymentSentinelLogsTable = () => {
  const [selectedLog, setSelectedLog] = useState<SentinelResponse | null>(null);
  const { logs, isLoading } = useDeploymentSentinelLogsQuery();

  return (
    <VirtualTable
      data={logs}
      isLoading={isLoading}
      columns={columns}
      onRowClick={setSelectedLog}
      selectedItem={selectedLog}
      keyExtractor={(log) => log.request_id}
      rowClassName={(log) => getRowClassName(log, selectedLog)}
      selectedClassName={getSelectedClassName}
      emptyState={
        <div className="w-full flex justify-center items-center h-full">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>Deployment Logs</Empty.Title>
            <Empty.Description className="text-left">
              No logs found for this deployment. Logs appear here when your deployment receives
              requests.
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

const WarningIcon = ({ status }: { status: number }) => (
  <TriangleWarning2
    iconSize="md-regular"
    className={cn(
      WARNING_ICON_STYLES.base,
      status < 300 && "invisible",
      status >= 400 && status < 500 && WARNING_ICON_STYLES.warning,
      status >= 500 && WARNING_ICON_STYLES.error,
    )}
  />
);

const columns: Column<SentinelResponse>[] = [
  {
    key: "time",
    header: "Time",
    width: "180px",
    headerClassName: "pl-8",
    render: (log) => (
      <div className="flex items-center gap-3 px-2">
        <WarningIcon status={log.response_status} />
        <div className="flex-1 min-w-0">
          <TimestampInfo
            value={log.time}
            className="font-mono group-hover:underline decoration-dotted"
          />
        </div>
      </div>
    ),
  },
  {
    key: "response_status",
    header: "Status",
    width: "120px",
    render: (log) => {
      const style = getStatusStyle(log.response_status);
      return (
        <Badge
          className={cn(
            "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
            style.badge.default,
          )}
        >
          {log.response_status}{" "}
          {extractResponseField(log, "code") ? `| ${extractResponseField(log, "code")}` : ""}
        </Badge>
      );
    },
  },
  {
    key: "host",
    header: "Hostname",
    width: "200px",
    render: (log) => (
      <div className="font-mono pr-4 truncate" title={log.host}>
        {log.host}
      </div>
    ),
  },
  {
    key: "method",
    header: "Method",
    width: "80px",
    render: (log) => (
      <Badge
        className={cn(
          "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
          getStatusStyle(log.response_status).badge.default,
        )}
      >
        {log.method}
      </Badge>
    ),
  },
  {
    key: "path",
    header: "Path",
    width: "250px",
    render: (log) => (
      <div className="font-mono pr-4 truncate" title={log.path}>
        {log.path}
      </div>
    ),
  },
  {
    key: "response_body",
    header: "Response Body",
    width: "300px",
    render: (log) => (
      <div className="font-mono whitespace-nowrap truncate max-w-[300px]" title={log.response_body}>
        {log.response_body}
      </div>
    ),
  },
  {
    key: "request_body",
    header: "Request Body",
    width: "1fr",
    render: (log) => (
      <div className="font-mono whitespace-nowrap truncate max-w-[300px]" title={log.request_body}>
        {log.request_body}
      </div>
    ),
  },
];
