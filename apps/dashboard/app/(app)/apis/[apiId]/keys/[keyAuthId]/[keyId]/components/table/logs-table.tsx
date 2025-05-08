"use client";
import { Badge } from "@/components/ui/badge";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import type { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import {
  Ban,
  BookBookmark,
  CircleCheck,
  Lock,
  ShieldKey,
  TimeClock,
  TriangleWarning2,
} from "@unkey/icons";
import { TimestampInfo } from "@unkey/ui";
import { Button, Empty, Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import { useState } from "react";
import { StatusBadge } from "./components/status-badge";
import { useKeyDetailsLogsQuery } from "./hooks/use-logs-query";

type LogOutcomeType = (typeof KEY_VERIFICATION_OUTCOMES)[number];
type LogOutcomeInfo = {
  type: LogOutcomeType;
  label: string;
  color: string;
  icon: React.ReactNode;
  tooltip: string;
};

const LOG_OUTCOME_DEFINITIONS: Record<LogOutcomeType, LogOutcomeInfo> = {
  VALID: {
    type: "VALID",
    label: "Valid",
    color: "bg-grayA-3 text-grayA-11 group-hover:bg-grayA-4",
    icon: <CircleCheck size="sm-regular" />,
    tooltip: "The key was successfully verified.",
  },
  INSUFFICIENT_PERMISSIONS: {
    type: "INSUFFICIENT_PERMISSIONS",
    label: "Unauthorized",
    color: "bg-errorA-3 text-errorA-11 group-hover:bg-errorA-5",
    icon: <Lock size="sm-regular" />,
    tooltip: "The key doesn't have sufficient permissions for this operation.",
  },
  RATE_LIMITED: {
    type: "RATE_LIMITED",
    label: "Ratelimited",
    color: "bg-warningA-3 text-warningA-11 group-hover:bg-warningA-4",
    icon: <TriangleWarning2 size="sm-regular" className="text-warning-11" />,
    tooltip: "The key has exceeded its rate limit.",
  },
  FORBIDDEN: {
    type: "FORBIDDEN",
    label: "Forbidden",
    color: "bg-errorA-3 text-errorA-11 group-hover:bg-errorA-4",
    icon: <Ban size="sm-regular" />,
    tooltip: "The key is not authorized for this operation.",
  },
  DISABLED: {
    type: "DISABLED",
    label: "Disabled",
    color: "bg-orangeA-3 text-orangeA-11 group-hover:bg-orangeA-4",
    icon: <ShieldKey size="sm-regular" />,
    tooltip: "The key has been disabled.",
  },
  EXPIRED: {
    type: "EXPIRED",
    label: "Expired",
    color: "bg-orangeA-3 text-orangeA-11 group-hover:bg-orangeA-4 ",
    icon: <TimeClock size="sm-regular" />,
    tooltip: "The key has expired and is no longer valid.",
  },
  USAGE_EXCEEDED: {
    type: "USAGE_EXCEEDED",
    label: "Usage Exceeded",
    color: "bg-errorA-3 text-errorA-11  group-hover:bg-errorA-4",
    icon: <TriangleWarning2 size="sm-regular" className="text-error-11" />,
    tooltip: "The key has exceeded its usage limit.",
  },
  "": {
    type: "",
    label: "Unknown",
    color: "bg-errorA-3 text-errorA-11 group-hover:bg-errorA-4",
    icon: <ShieldKey size="sm-regular" className="text-gray-11" />,
    tooltip: "Unknown verification status.",
  },
};

export type StatusStyle = {
  base: string;
  hover: string;
  selected: string;
  badge?: {
    default: string;
    selected: string;
  };
  focusRing: string;
};

export const STATUS_STYLES = {
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
    hover: "hover:text-warning-11 hover:bg-warning-3",
    selected: "text-warning-11 bg-warning-3",
    focusRing: "focus:ring-warning-7",
  },
  blocked: {
    base: "text-orange-11 bg-orange-2",
    hover: "hover:text-orange-11 hover:bg-orange-3",
    selected: "text-orange-11 bg-orange-3",
    focusRing: "focus:ring-orange-7",
  },
  error: {
    base: "text-error-11 bg-error-2",
    hover: "hover:text-error-11 hover:bg-error-3",
    selected: "text-error-11 bg-error-3",
    focusRing: "focus:ring-error-7",
  },
};

export const categorizeSeverity = (outcome: string): keyof typeof STATUS_STYLES => {
  switch (outcome) {
    case "VALID":
      return "success";
    case "RATE_LIMITED":
      return "warning";
    case "DISABLED":
    case "EXPIRED":
      return "blocked";
    case "INSUFFICIENT_PERMISSIONS":
    case "FORBIDDEN":
    case "USAGE_EXCEEDED":
      return "error";
    default:
      return "success";
  }
};

export const getStatusStyle = (log: KeyDetailsLog): StatusStyle => {
  const severity = categorizeSeverity(log.outcome);
  return STATUS_STYLES[severity];
};

type Props = {
  keyId: string;
  apiId: string;
  keyspaceId: string;
};

export const KeyDetailsLogsTable = ({ apiId, keyspaceId, keyId }: Props) => {
  const [selectedLog, setSelectedLog] = useState<KeyDetailsLog | null>(null);

  const { logs, isLoading, isLoadingMore, loadMore, totalCount, hasMore } = useKeyDetailsLogsQuery({
    apiId,
    keyId,
    keyspaceId,
  });

  const getStatusProperties = (outcome: string) => {
    const outcomeType =
      (outcome as LogOutcomeType) in LOG_OUTCOME_DEFINITIONS ? (outcome as LogOutcomeType) : "";

    const outcomeInfo = LOG_OUTCOME_DEFINITIONS[outcomeType];

    return {
      primary: {
        label: outcomeInfo.label,
        color: outcomeInfo.color,
        icon: outcomeInfo.icon,
      },
      count: 0,
    };
  };

  const getRowClassName = (log: KeyDetailsLog, selected: KeyDetailsLog | null) => {
    const style = getStatusStyle(log);
    const isSelected = selected?.request_id === log.request_id;

    return cn(
      style.base,
      style.hover,
      "group rounded-md cursor-pointer transition-colors",
      "focus:outline-none focus:ring-1 focus:ring-opacity-40",
      style.focusRing,
      isSelected && style.selected,
    );
  };

  const columns = (): Column<KeyDetailsLog>[] => {
    return [
      {
        key: "time",
        header: "Time",
        width: "5%",
        headerClassName: "pl-2",
        render: (log) => (
          <TimestampInfo
            value={log.time}
            className={cn(
              "font-mono group-hover:underline decoration-dotted pl-2",
              selectedLog && selectedLog.request_id !== log.request_id && "pointer-events-none",
            )}
          />
        ),
      },
      {
        key: "outcome",
        header: "Status",
        width: "15%",
        render: (log) => {
          const statusProps = getStatusProperties(log.outcome);
          const outcomeType =
            (log.outcome as LogOutcomeType) in LOG_OUTCOME_DEFINITIONS
              ? (log.outcome as LogOutcomeType)
              : "";
          const outcomeInfo = LOG_OUTCOME_DEFINITIONS[outcomeType];
          return (
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger className="cursor-default">
                  <div className="flex gap-3 items-center">
                    <StatusBadge primary={statusProps.primary} />
                  </div>
                </TooltipTrigger>
                <TooltipContent>
                  <p>{outcomeInfo.tooltip}</p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          );
        },
      },
      {
        key: "region",
        header: "Region",
        width: "15%",
        render: (log) => (
          <div className="flex items-center uppercase font-mono">
            <div className="w-full max-w-[65px] truncate whitespace-nowrap" title={log.region}>
              {log.region}
            </div>
          </div>
        ),
      },
      {
        key: "tags",
        header: "Tags",
        width: "20%",
        render: (log) => {
          return (
            <div className="flex flex-wrap gap-1 items-center">
              {log.tags && log.tags.length > 0 ? (
                log.tags.slice(0, 3).map((tag) => (
                  <Badge
                    key={tag}
                    title={tag}
                    className={cn(
                      "px-[6px] rounded-md whitespace-nowrap",
                      selectedLog?.request_id === log.request_id
                        ? STATUS_STYLES.success.badge.selected
                        : STATUS_STYLES.success.badge.default,
                    )}
                  >
                    {tag.length > 15 ? `${tag.substring(0, 12)}...` : tag}
                  </Badge>
                ))
              ) : (
                <span className="text-gray-8">—</span>
              )}
              {log.tags && log.tags.length > 3 && (
                <Badge
                  className={cn(
                    "px-[6px] rounded-md whitespace-nowrap",
                    selectedLog?.request_id === log.request_id
                      ? STATUS_STYLES.success.badge.selected
                      : STATUS_STYLES.success.badge.default,
                  )}
                >
                  +{log.tags.length - 3}
                </Badge>
              )}
            </div>
          );
        },
      },
    ];
  };

  return (
    <div className="flex flex-col">
      <VirtualTable
        data={logs}
        isLoading={isLoading}
        isFetchingNextPage={isLoadingMore}
        onLoadMore={loadMore}
        columns={columns()}
        onRowClick={setSelectedLog}
        selectedItem={selectedLog}
        keyExtractor={(log) => log.request_id}
        rowClassName={(log) => getRowClassName(log, selectedLog)}
        loadMoreFooterProps={{
          buttonText: "Load more logs",
          hasMore,
          hide: isLoading,
          countInfoText: (
            <div className="flex gap-2">
              <span>Showing</span> <span className="text-accent-12">{logs.length}</span>
              <span>of</span>
              {totalCount}
              <span>requests</span>
            </div>
          ),
        }}
        emptyState={
          <div className="w-full flex justify-center items-center h-full">
            <Empty className="w-[400px] flex items-start">
              <Empty.Icon className="w-auto" />
              <Empty.Title>Key Verification Logs</Empty.Title>
              <Empty.Description className="text-left">
                No verification logs found for this key. When this API key is used, details about
                each verification attempt will appear here.
              </Empty.Description>
              <Empty.Actions className="mt-4 justify-center md:justify-start">
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
    </div>
  );
};
