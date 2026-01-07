"use client";

import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { useQueryTime } from "@/providers/query-time-provider";
import { Badge, Button, CopyButton, Empty, InfoTooltip, TimestampInfo } from "@unkey/ui";
import { BookBookmark } from "@unkey/icons";
import { useCallback, useState } from "react";
import { 
  LOG_OUTCOME_DEFINITIONS, 
  STATUS_STYLES, 
  getOutcomeInfo, 
  getStatusType, 
  categorizeSeverity,
  type LogOutcomeType,
  type StatusStyle
} from "@/components/logs/verification-outcomes/outcome-definitions";
import { StatusBadge } from "./components/status-badge";

export type VerificationLog = {
  request_id: string;
  time: number;
  outcome: string;
  region?: string;
  tags?: string[];
  [key: string]: any;
};

export type DataHook<TLog extends VerificationLog> = () => {
  realtimeLogs: TLog[];
  historicalLogs: TLog[];
  isLoading: boolean;
  isLoadingMore: boolean;
  loadMore: () => void;
  hasMore: boolean;
  totalCount: number;
};

export type EmptyStateConfig = {
  title: string;
  description: string;
  actions?: React.ReactNode;
};

export interface VerificationLogsTableProps<TLog extends VerificationLog> {
  // Injected data fetching
  dataHook: DataHook<TLog>;
  // Selection handling
  selectedLog: TLog | null;
  onLogSelect: (log: TLog | null) => void;
  // Configuration
  columns: Column<TLog>[];
  emptyState: EmptyStateConfig;
  // Row interaction handlers
  onRowMouseEnter?: (log: TLog) => void;
  onRowMouseLeave?: () => void;
}

export const getStatusStyle = (log: VerificationLog): StatusStyle => {
  const severity = categorizeSeverity(log.outcome);
  return STATUS_STYLES[severity];
};

export function VerificationLogsTable<TLog extends VerificationLog>({
  dataHook,
  selectedLog,
  onLogSelect,
  columns,
  emptyState,
  onRowMouseEnter,
  onRowMouseLeave,
}: VerificationLogsTableProps<TLog>) {
  const { realtimeLogs, historicalLogs, isLoading, isLoadingMore, loadMore, hasMore, totalCount } =
    dataHook();

  const getRowClassName = (log: TLog, selected: TLog | null) => {
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
    (log: TLog) => {
      if (log.request_id !== hoveredLogId) {
        setHoveredLogId(log.request_id);

        // Prefetch log details for better UX
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
      
      // Call the injected handler if provided
      if (onRowMouseEnter) {
        onRowMouseEnter(log);
      }
    },
    [hoveredLogId, utils.logs.queryLogs, timestamp, onRowMouseEnter],
  );

  const handleRowMouseLeave = useCallback(() => {
    setHoveredLogId(null);
    if (onRowMouseLeave) {
      onRowMouseLeave();
    }
  }, [onRowMouseLeave]);

  return (
    <div className="flex flex-col">
      <VirtualTable
        data={historicalLogs}
        realtimeData={realtimeLogs}
        isLoading={isLoading}
        isFetchingNextPage={isLoadingMore}
        onLoadMore={loadMore}
        columns={columns}
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
              <span>Showing</span>{" "}
              <span className="text-accent-12">
                {new Intl.NumberFormat().format(historicalLogs.length)}
              </span>
              <span>of</span>
              {new Intl.NumberFormat().format(totalCount)}
              <span>requests</span>
            </div>
          ),
        }}
        emptyState={
          <div className="w-full flex justify-center items-center h-full">
            <Empty className="w-[400px] flex items-start">
              <Empty.Icon className="w-auto" />
              <Empty.Title>{emptyState.title}</Empty.Title>
              <Empty.Description className="text-left">
                {emptyState.description}
              </Empty.Description>
              {emptyState.actions && (
                <Empty.Actions className="mt-4 justify-center md:justify-start">
                  {emptyState.actions}
                </Empty.Actions>
              )}
            </Empty>
          </div>
        }
      />
    </div>
  );
}

// Utility function to create common verification log columns
export function createBaseVerificationColumns<TLog extends VerificationLog>(
  selectedLog: TLog | null
): Column<TLog>[] {
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
        const outcomeInfo = getOutcomeInfo(log.outcome);
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
}