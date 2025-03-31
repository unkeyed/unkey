"use client";
import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { VirtualTable, type VirtualTableRef } from "@/components/virtual-table";
import type { Column } from "@/components/virtual-table/types";
import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { cn } from "@unkey/ui/src/lib/utils";
import { FunctionSquare, KeySquare } from "lucide-react";
import { useEffect, useRef } from "react";
import { useAuditLogsQuery } from "./hooks/use-logs-query";
import {
  getAuditRowClassName,
  getAuditSelectedClassName,
  getAuditStatusStyle,
  getEventType,
} from "./utils/get-row-class";

type Props = {
  selectedLog: AuditLog | null;
  setSelectedLog: (log: AuditLog | null) => void;
  onMount: (distanceToTop: number) => void;
};

export const AuditLogsTable = ({ selectedLog, setSelectedLog, onMount }: Props) => {
  const tableRef = useRef<VirtualTableRef>(null);
  const { historicalLogs, loadMore, isLoadingMore, isLoading } = useAuditLogsQuery({});

  useEffect(() => {
    const distanceToTop = tableRef.current?.containerRef?.getBoundingClientRect().top ?? 0;
    onMount(distanceToTop);
  }, [onMount]);

  return (
    <VirtualTable
      ref={tableRef}
      data={historicalLogs}
      columns={columns(selectedLog)}
      isLoading={isLoading}
      isFetchingNextPage={isLoadingMore}
      onLoadMore={loadMore}
      rowClassName={(log) =>
        getAuditRowClassName(
          log,
          selectedLog?.auditLog.id === log.auditLog.id,
          Boolean(selectedLog),
        )
      }
      selectedItem={selectedLog}
      onRowClick={setSelectedLog}
      selectedClassName={getAuditSelectedClassName}
      keyExtractor={(log) => log.auditLog.id}
      config={{
        loadingRows: 50,
      }}
    />
  );
};

export const columns = (selectedLog: AuditLog | null): Column<AuditLog>[] => [
  {
    key: "time",
    header: "Time",
    width: "150px",
    headerClassName: "pl-2",
    render: (log) => {
      return (
        <TimestampInfo
          value={log.auditLog.time}
          className={cn(
            "font-mono group-hover:underline decoration-dotted pl-2",
            selectedLog && selectedLog.auditLog.id !== log.auditLog.id && "pointer-events-none",
          )}
        />
      );
    },
  },
  {
    key: "actor",
    header: "Actor",
    width: "15%",
    render: (log) => (
      <div className="flex items-center gap-3 truncate">
        {log.auditLog.actor.type === "user" && log.user ? (
          <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs">
            <span className="text-xs whitespace-nowrap">
              {`${log.user.firstName ?? ""} ${log.user.lastName ?? ""}`}
            </span>
          </div>
        ) : log.auditLog.actor.type === "key" ? (
          <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs">
            <KeySquare className="w-4 h-4" />
            <span className="font-mono text-xs truncate">{log.auditLog.actor.id}</span>
          </div>
        ) : (
          <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs">
            <FunctionSquare className="w-4 h-4" />
            <span className="font-mono text-xs truncate">{log.auditLog.actor.id}</span>
          </div>
        )}
      </div>
    ),
  },
  {
    key: "action",
    header: "Action",
    width: "15%",
    render: (log) => {
      const eventType = getEventType(log.auditLog.event);
      const style = getAuditStatusStyle(log);
      const isSelected = log.auditLog.id === selectedLog?.auditLog.id;

      return (
        <div className="flex items-center gap-3 group/action">
          <Badge
            className={cn(
              "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
              isSelected ? style.badge.selected : style.badge.default,
            )}
          >
            {eventType}
          </Badge>
        </div>
      );
    },
  },
  {
    key: "event",
    header: "Event",
    width: "20%",
    render: (log) => (
      <div className="flex items-center gap-2 font-mono text-xs truncate">
        <span>{log.auditLog.event}</span>
      </div>
    ),
  },
  {
    key: "event-description",
    header: "Description",
    width: "auto",
    render: (log) => (
      <div className="font-mono text-xs truncate w-[200px]">{log.auditLog.description}</div>
    ),
  },
];
