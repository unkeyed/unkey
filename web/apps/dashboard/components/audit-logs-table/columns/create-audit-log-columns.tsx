import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { cn } from "@/lib/utils";
import { Key, MathFunction } from "@unkey/icons";
import type { DataTableColumnDef } from "@unkey/ui";
import { Badge, TimestampInfo } from "@unkey/ui";
import { getAuditStatusStyle, getEventType } from "../utils/get-row-class";

export const AUDIT_LOG_COLUMN_IDS = {
  TIME: { id: "time", header: "Time" },
  ACTOR: { id: "actor", header: "Actor" },
  ACTION: { id: "action", header: "Action" },
  EVENT: { id: "event", header: "Event" },
  DESCRIPTION: { id: "description", header: "Description" },
} as const;

type CreateAuditLogColumnsOptions = {
  selectedLog: AuditLog | null;
};

export const createAuditLogColumns = ({
  selectedLog,
}: CreateAuditLogColumnsOptions): DataTableColumnDef<AuditLog>[] => [
  {
    id: AUDIT_LOG_COLUMN_IDS.TIME.id,
    header: AUDIT_LOG_COLUMN_IDS.TIME.header,
    enableSorting: false,
    meta: {
      width: "12%",
      headerClassName: "pl-2",
    },
    cell: ({ row }) => {
      const log = row.original;
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
    id: AUDIT_LOG_COLUMN_IDS.ACTOR.id,
    header: AUDIT_LOG_COLUMN_IDS.ACTOR.header,
    enableSorting: false,
    meta: {
      width: "15%",
    },
    cell: ({ row }) => {
      const log = row.original;
      return (
        <div className="flex items-center gap-3 truncate">
          {log.auditLog.actor.type === "user" && log.user ? (
            <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs">
              <span className="text-xs whitespace-nowrap secret truncate">
                {`${log.user.firstName ?? ""} ${log.user.lastName ?? ""}`}
              </span>
            </div>
          ) : log.auditLog.actor.type === "key" ? (
            <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs">
              <Key iconSize="sm-thin" />
              <span className="font-mono text-xs truncate secret">{log.auditLog.actor.id}</span>
            </div>
          ) : (
            <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs">
              <MathFunction iconSize="sm-thin" />
              <span className="font-mono text-xs truncate secret">{log.auditLog.actor.id}</span>
            </div>
          )}
        </div>
      );
    },
  },
  {
    id: AUDIT_LOG_COLUMN_IDS.ACTION.id,
    header: AUDIT_LOG_COLUMN_IDS.ACTION.header,
    enableSorting: false,
    meta: {
      width: "10%",
    },
    cell: ({ row }) => {
      const log = row.original;
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
    id: AUDIT_LOG_COLUMN_IDS.EVENT.id,
    header: AUDIT_LOG_COLUMN_IDS.EVENT.header,
    enableSorting: false,
    meta: {
      width: "25%",
    },
    cell: ({ row }) => {
      const log = row.original;
      return (
        <div className="flex items-center gap-2 font-mono text-xs overflow-hidden">
          <span className="truncate">{log.auditLog.event}</span>
        </div>
      );
    },
  },
  {
    id: AUDIT_LOG_COLUMN_IDS.DESCRIPTION.id,
    header: AUDIT_LOG_COLUMN_IDS.DESCRIPTION.header,
    enableSorting: false,
    meta: {
      width: "38%",
    },
    cell: ({ row }) => {
      const log = row.original;
      return (
        <div className="font-mono text-xs overflow-hidden text-ellipsis whitespace-nowrap truncate min-w-0">
          {log.auditLog.description}
        </div>
      );
    },
  },
];
