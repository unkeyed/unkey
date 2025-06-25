"use client";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { useQueryTime } from "@/providers/query-time-provider";
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
import { Badge, Button, CopyButton, Empty, InfoTooltip, TimestampInfo } from "@unkey/ui";
import { useCallback, useState } from "react";
import { useKeyDetailsLogsContext } from "../../context/logs";
import { StatusBadge } from "./components/status-badge";
import { useKeyDetailsLogsQuery } from "./hooks/use-logs-query";

type LogOutcomeType = (typeof KEY_VERIFICATION_OUTCOMES)[number];
type LogOutcomeInfo = {
  type: LogOutcomeType;
  label: string;
  icon: React.ReactNode;
  tooltip: string;
};

const LOG_OUTCOME_DEFINITIONS: Record<LogOutcomeType, LogOutcomeInfo> = {
  VALID: {
    type: "VALID",
    label: "Valid",
    icon: <CircleCheck size="sm-regular" />,
    tooltip: "The key was successfully verified.",
  },
  INSUFFICIENT_PERMISSIONS: {
    type: "INSUFFICIENT_PERMISSIONS",
    label: "Unauthorized",
    icon: <Lock size="sm-regular" className="text-errorA-11" />,
    tooltip: "The key doesn't have sufficient permissions for this operation.",
  },
  RATE_LIMITED: {
    type: "RATE_LIMITED",
    label: "Ratelimited",
    icon: <TriangleWarning2 size="sm-regular" className="text-warningA-11" />,
    tooltip: "The key has exceeded its rate limit.",
  },
  FORBIDDEN: {
    type: "FORBIDDEN",
    label: "Forbidden",
    icon: <Ban size="sm-regular" className="text-errorA-11" />,
    tooltip: "The key is not authorized for this operation.",
  },
  DISABLED: {
    type: "DISABLED",
    label: "Disabled",
    icon: <ShieldKey size="sm-regular" className="text-orangeA-11" />,
    tooltip: "The key has been disabled.",
  },
  EXPIRED: {
    type: "EXPIRED",
    label: "Expired",
    icon: <TimeClock size="sm-regular" className="text-orangeA-11" />,
    tooltip: "The key has expired and is no longer valid.",
  },
  USAGE_EXCEEDED: {
    type: "USAGE_EXCEEDED",
    label: "Usage Exceeded",
    icon: <TriangleWarning2 size="sm-regular" className="text-errorA-11" />,
    tooltip: "The key has exceeded its usage limit.",
  },
  "": {
    type: "",
    label: "Unknown",
    icon: <ShieldKey size="sm-regular" />,
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
    hover: "hover:text-grayA-11 hover:bg-grayA-3 dark:hover:text-grayA-12",
    selected: "text-grayA-12 bg-grayA-3",
    badge: {
      default: "bg-grayA-3 text-grayA-9 group-hover:bg-grayA-4",
      selected: "bg-grayA-4 text-grayA-11",
    },
    focusRing: "focus:ring-accent-7",
  },
  warning: {
    base: "text-warningA-11 bg-warning-2",
    hover: "hover:text-warningA-11 hover:bg-warningA-3",
    selected: "text-warningA-11 bg-warningA-3",
    badge: {
      default: "bg-warningA-3 text-warningA-11 group-hover:bg-warningA-4",
      selected: "bg-warningA-4 text-warningA-11",
    },
    focusRing: "focus:ring-warning-7",
  },
  blocked: {
    base: "text-orangeA-11 bg-orange-2",
    hover: "hover:text-orangeA-11 hover:bg-orangeA-3",
    selected: "text-orangeA-11 bg-orangeA-3",
    badge: {
      default: "bg-orangeA-3 text-orangeA-11 group-hover:bg-orangeA-4",
      selected: "bg-orangeA-4 text-orangeA-11",
    },
    focusRing: "focus:ring-orange-7",
  },
  error: {
    base: "text-errorA-11 bg-error-2",
    hover: "hover:text-errorA-11 hover:bg-errorA-3",
    selected: "text-errorA-11 bg-errorA-3",
    badge: {
      default: "bg-errorA-3 text-errorA-11 group-hover:bg-errorA-4",
      selected: "bg-errorA-4 text-errorA-11",
    },
    focusRing: "focus:ring-error-7",
  },
};

const getStatusType = (outcome: LogOutcomeType): keyof typeof STATUS_STYLES => {
  switch (outcome) {
    case "VALID":
      return "success";
    case "RATE_LIMITED":
      return "warning";
    case "DISABLED":
    case "EXPIRED":
      return "blocked";
    default:
      return "error";
  }
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
  keyspaceId: string;
  selectedLog: KeyDetailsLog | null;
  onLogSelect: (log: KeyDetailsLog | null) => void;
};

export const KeyDetailsLogsTable = ({ keyspaceId, keyId, selectedLog, onLogSelect }: Props) => {
  const { isLive } = useKeyDetailsLogsContext();
  const { realtimeLogs, historicalLogs, isLoading, isLoadingMore, loadMore, hasMore, totalCount } =
    useKeyDetailsLogsQuery({
      keyId,
      keyspaceId,
      startPolling: isLive,
      pollIntervalMs: 2000,
    });

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

  const [hoveredLogId, setHoveredLogId] = useState<string | null>(null);
  const { queryTime: timestamp } = useQueryTime();
  const utils = trpc.useUtils();

  const handleRowHover = useCallback(
    (log: KeyDetailsLog) => {
      if (log.request_id !== hoveredLogId) {
        setHoveredLogId(log.request_id);

        utils.logs.queryLogs.prefetch({
          limit: 1,
          startTime: 0,
          endTime: timestamp,
          host: { filters: [] },
          method: { filters: [] },
          path: { filters: [] },
          status: { filters: [] },
          requestId: {
            filters: [
              {
                operator: "is",
                value: log.request_id,
              },
            ],
          },
          since: "",
        });
      }
    },
    [hoveredLogId, utils.logs.queryLogs, timestamp],
  );

  const handleRowMouseLeave = useCallback(() => {
    setHoveredLogId(null);
  }, []);

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
          const isSelected = selectedLog?.request_id === log.request_id;
          const outcomeType =
            (log.outcome as LogOutcomeType) in LOG_OUTCOME_DEFINITIONS
              ? (log.outcome as LogOutcomeType)
              : "";
          const outcomeInfo = LOG_OUTCOME_DEFINITIONS[outcomeType];
          return (
            <InfoTooltip
              variant="inverted"
              className="cursor-default"
              content={<p>{outcomeInfo.tooltip}</p>}
              position={{ side: "top", align: "center", sideOffset: 5 }}
            >
              <div className="flex gap-3 items-center">
                <StatusBadge
                  primary={{
                    label: outcomeInfo.label,
                    color: isSelected
                      ? STATUS_STYLES[getStatusType(outcomeInfo.type)].badge.selected
                      : STATUS_STYLES[getStatusType(outcomeInfo.type)].badge.default,
                    icon: outcomeInfo.icon,
                  }}
                />
              </div>
            </InfoTooltip>
          );
        },
      },
      {
        key: "region",
        header: "Region",
        width: "15%",
        render: (log) => (
          <div className="flex items-center font-mono">
            <div className="w-full whitespace-nowrap" title={log.region}>
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
                  <InfoTooltip
                    variant="inverted"
                    className="px-2 py-1"
                    key={tag}
                    content={
                      <div className="max-w-xs">
                        {tag.length > 60 ? (
                          <div>
                            <div className="break-all max-w-[300px] truncate">{tag}</div>
                            <div className="flex items-center justify-between mt-1.5">
                              <div className="text-xs opacity-60">({tag.length} characters)</div>
                              {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
                              <div
                                className="pointer-events-auto"
                                onClick={(e) => e.stopPropagation()}
                              >
                                <CopyButton variant="ghost" value={tag} />
                              </div>
                            </div>
                          </div>
                        ) : (
                          <div className="flex justify-between items-center gap-1.5">
                            <div className="break-all max-w-[300px] truncate">{tag}</div>
                            {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
                            <div
                              className="pointer-events-auto flex-shrink-0"
                              onClick={(e) => e.stopPropagation()}
                            >
                              <CopyButton variant="ghost" value={tag} />
                            </div>
                          </div>
                        )}
                      </div>
                    }
                    position={{ side: "top", align: "start", sideOffset: 5 }}
                    asChild
                  >
                    <Badge
                      className={cn(
                        "whitespace-nowrap max-w-[150px] truncate",
                        selectedLog?.request_id === log.request_id
                          ? STATUS_STYLES.success.badge.selected
                          : "",
                      )}
                    >
                      {shortenId(tag, {
                        endChars: 0,
                        minLength: 14,
                        startChars: 10,
                      })}
                    </Badge>
                  </InfoTooltip>
                ))
              ) : (
                <span className="text-gray-8">—</span>
              )}
              {log.tags && log.tags.length > 3 && (
                <InfoTooltip
                  variant="inverted"
                  content={
                    <div className="flex flex-col gap-2 py-1 max-w-xs max-h-[300px] overflow-y-auto">
                      <div className="text-xs opacity-75 font-medium">
                        {log.tags.length - 3} more tags:
                      </div>
                      {log.tags.slice(3).map((tag, idx) => (
                        <div key={idx + tag} className="text-xs">
                          {tag.length > 60 ? (
                            <div>
                              <div className="break-all max-w-[300px] truncate">{tag}</div>
                              <div className="flex items-center justify-between mt-1.5">
                                <div className="text-xs opacity-60">({tag.length} characters)</div>
                                {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
                                <div
                                  className="pointer-events-auto"
                                  onClick={(e) => e.stopPropagation()}
                                >
                                  <CopyButton variant="ghost" value={tag} />
                                </div>
                              </div>
                            </div>
                          ) : (
                            <div className="flex justify-between items-start gap-1.5">
                              <div className="break-all max-w-[300px] truncate">{tag}</div>
                              {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
                              <div
                                className="pointer-events-auto flex-shrink-0"
                                onClick={(e) => e.stopPropagation()}
                              >
                                <CopyButton variant="ghost" value={tag} />
                              </div>
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  }
                  position={{ side: "top", align: "start", sideOffset: 5 }}
                  asChild
                >
                  <Badge
                    className={cn(
                      "whitespace-nowrap",
                      selectedLog?.request_id === log.request_id
                        ? STATUS_STYLES.success.badge.selected
                        : "",
                    )}
                  >
                    +{log.tags.length - 3}
                  </Badge>
                </InfoTooltip>
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
        data={historicalLogs}
        realtimeData={realtimeLogs}
        isLoading={isLoading}
        isFetchingNextPage={isLoadingMore}
        onLoadMore={loadMore}
        columns={columns()}
        onRowClick={onLogSelect}
        onRowMouseEnter={handleRowHover}
        onRowMouseLeave={handleRowMouseLeave}
        selectedItem={selectedLog}
        keyExtractor={(log) => log.request_id}
        rowClassName={(log) => getRowClassName(log, selectedLog)}
        loadMoreFooterProps={{
          buttonText: "Load more logs",
          hasMore,
          hide: isLoading,
          countInfoText: (
            <div className="flex gap-2">
              <span>Showing</span> <span className="text-accent-12">{historicalLogs.length}</span>
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
