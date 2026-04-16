import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { cn } from "@/lib/utils";
import type { DataTableColumnDef } from "@unkey/ui";
import { TimestampInfo } from "@unkey/ui";
import { MonoTextCell } from "@unkey/ui";
import { ActorCell } from "../components/cells/actor-cell";
import { AuditActionBadgeCell } from "../components/cells/audit-action-badge-cell";

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
      return <ActorCell log={row.original} />;
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
      const isSelected = log.auditLog.id === selectedLog?.auditLog.id;
      return <AuditActionBadgeCell log={log} isSelected={isSelected} />;
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
      return <MonoTextCell value={row.original.auditLog.event} />;
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
      return <MonoTextCell value={row.original.auditLog.description} />;
    },
  },
];
