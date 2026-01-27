"use client";

import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import type { SentinelResponse } from "@unkey/clickhouse/src/sentinel";
import { BookBookmark, TriangleWarning2 } from "@unkey/icons";
import { Badge, Button, Empty, TimestampInfo } from "@unkey/ui";
import {
  WARNING_ICON_STYLES,
  getStatusStyle,
} from "../../../sentinel-logs/components/table/utils/get-row-class";
import { useDeploymentSentinelLogsQuery } from "../hooks/use-deployment-sentinel-logs-query";
import {
  formatLatency,
  getLatencyStyle,
} from "./deployment-sentinel-logs-table.utils";


const getRowClassName = (log: SentinelResponse): string => {
  if (!log?.request_id) {
    throw new Error("Log must have a valid request_id");
  }

  const style = getStatusStyle(log.response_status);

  const baseClasses = [
    style.base,
    style.hover,
    "group rounded-md",
    style.focusRing,
  ];
  return cn(...baseClasses);
};

export const DeploymentSentinelLogsTable = () => {
  const { logs, isLoading } = useDeploymentSentinelLogsQuery();

  return (
    <VirtualTable
      data={logs}
      isLoading={isLoading}
      columns={columns}
      keyExtractor={(log) => log.request_id}
      rowClassName={(log) => getRowClassName(log)}
      className="h-full"
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


const LatencyBadge = ({ log }: { log: SentinelResponse }) => {
  const style = getLatencyStyle(log.total_latency);
  const tooltipText = `Total: ${log.total_latency}ms | Instance: ${log.instance_latency}ms | Sentinel: ${log.sentinel_latency}ms`;

  return (
    <Badge
      className={cn("px-[6px] rounded-md font-mono whitespace-nowrap tabular-nums", style)}
      title={tooltipText}
    >
      {formatLatency(log.total_latency)}
    </Badge>
  );
};

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
          {log.response_status}
        </Badge>
      );
    },
  },
  {
    key: "region",
    header: "Region",
    width: "100px",
    render: (log) => (
      <div className="font-mono pr-4 truncate uppercase" title={log.region}>
        {log.region}
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
    key: "ip_address",
    header: "IP Address",
    width: "140px",
    render: (log) => (
      <div className="font-mono pr-4 truncate" title={log.ip_address}>
        {log.ip_address}
      </div>
    ),
  },
  {
    key: "user_agent",
    header: "User Agent",
    width: "200px",
    render: (log) => (
      <div className="font-mono pr-4 truncate" title={log.user_agent}>
        {log.user_agent}
      </div>
    ),
  },
  {
    key: "latency",
    header: "Latency",
    width: "150px",
    render: (log) => <LatencyBadge log={log} />,
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
