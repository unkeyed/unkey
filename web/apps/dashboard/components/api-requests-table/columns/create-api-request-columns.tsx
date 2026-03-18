"use client";
import { formatNumber } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import {
  Badge,
  type DataTableColumnDef,
  InvalidCountCell,
  SortableHeader,
  TimestampInfo,
} from "@unkey/ui";
import { KeyIdentifierColumn } from "../components/key-identifier-column";
import { getErrorPercentage, getSuccessPercentage } from "../utils/calculate-blocked-percentage";
import { SEVERITY_STYLES, getStatusStyle } from "../utils/get-row-class";

const TruncatedTextCell = ({ value }: { value: string }) => (
  <div className="flex items-center font-mono">
    <div className="w-full max-w-37.5 truncate whitespace-nowrap" title={value}>
      {value}
    </div>
  </div>
);

export const API_REQUEST_COLUMN_IDS = {
  KEY_ID: { id: "key_id", accessorKey: "key_id", header: "ID" },
  NAME: { id: "name", accessorKey: "name", header: "Name" },
  EXTERNAL_ID: { id: "external_id", accessorKey: "external_id", header: "External ID" },
  VALID: { id: "valid", accessorKey: "valid", header: "Valid" },
  INVALID: { id: "invalid", accessorKey: "invalid", header: "Invalid" },
  LAST_USED: { id: "lastUsed", accessorKey: "lastUsed", header: "Last Used" },
} as const;

type CreateLogsColumnsOptions = {
  selectedLog: KeysOverviewLog | null;
  apiId: string;
  onNavigate?: () => void;
};

export const createApiRequestColumns = ({
  selectedLog,
  apiId,
  onNavigate,
}: CreateLogsColumnsOptions): DataTableColumnDef<KeysOverviewLog>[] => [
  {
    id: API_REQUEST_COLUMN_IDS.KEY_ID.id,
    accessorKey: API_REQUEST_COLUMN_IDS.KEY_ID.accessorKey,
    header: API_REQUEST_COLUMN_IDS.KEY_ID.header,
    enableSorting: false,
    meta: {
      width: "15%",
      headerClassName: "pl-12",
    },
    cell: ({ row }) => (
      <KeyIdentifierColumn log={row.original} apiId={apiId} onNavigate={onNavigate} />
    ),
  },
  {
    id: API_REQUEST_COLUMN_IDS.NAME.id,
    header: API_REQUEST_COLUMN_IDS.NAME.header,
    enableSorting: false,
    meta: {
      width: "15%",
    },
    cell: ({ row }) => <TruncatedTextCell value={row.original.key_details?.name || "—"} />,
  },
  {
    id: API_REQUEST_COLUMN_IDS.EXTERNAL_ID.id,
    header: API_REQUEST_COLUMN_IDS.EXTERNAL_ID.header,
    enableSorting: false,
    meta: {
      width: "15%",
    },
    cell: ({ row }) => (
      <TruncatedTextCell
        value={
          (row.original.key_details?.identity?.external_id ?? row.original.key_details?.owner_id) ||
          "—"
        }
      />
    ),
  },
  {
    id: API_REQUEST_COLUMN_IDS.VALID.id,
    accessorKey: API_REQUEST_COLUMN_IDS.VALID.accessorKey,
    header: ({ header }) => (
      <SortableHeader key={API_REQUEST_COLUMN_IDS.VALID.id} header={header}>
        {API_REQUEST_COLUMN_IDS.VALID.header}
      </SortableHeader>
    ),
    meta: {
      width: "15%",
    },
    cell: ({ row }) => {
      const log = row.original;
      const successPercentage = getSuccessPercentage(log);
      return (
        <div className="flex gap-3 items-center tabular-nums">
          <Badge
            className={cn(
              "px-[1.5 rounded-md font-mono whitespace-nowrap",
              selectedLog?.key_id === log.key_id
                ? SEVERITY_STYLES.success.badge.selected
                : SEVERITY_STYLES.success.badge.default,
            )}
            title={`${log.valid_count.toLocaleString()} Valid requests (${successPercentage.toFixed(1)}%)`}
          >
            {formatNumber(log.valid_count)}
          </Badge>
        </div>
      );
    },
  },
  {
    id: "invalid",
    accessorKey: "error_count",
    header: ({ header }) => (
      <SortableHeader key={API_REQUEST_COLUMN_IDS.INVALID.id} header={header}>
        {API_REQUEST_COLUMN_IDS.INVALID.header}
      </SortableHeader>
    ),
    meta: {
      width: "15%",
    },
    cell: ({ row }) => {
      const log = row.original;
      const style = getStatusStyle(log);
      const errorPercentage = getErrorPercentage(log);
      return (
        <InvalidCountCell
          count={log.error_count}
          outcomeCounts={log.outcome_counts}
          isSelected={selectedLog?.key_id === log.key_id}
          badgeClassName={
            selectedLog?.key_id === log.key_id ? style.badge.selected : style.badge.default
          }
          title={`${log.error_count.toLocaleString()} Invalid requests (${errorPercentage.toFixed(1)}%)`}
        />
      );
    },
  },
  {
    id: "time",
    accessorKey: "time",
    header: ({ header }) => (
      <SortableHeader key={API_REQUEST_COLUMN_IDS.LAST_USED.id} header={header}>
        {API_REQUEST_COLUMN_IDS.LAST_USED.header}
      </SortableHeader>
    ),
    meta: {
      width: "15%",
    },
    cell: ({ row }) => {
      const log = row.original;
      return (
        <TimestampInfo
          value={log.time}
          className={cn(
            "font-mono group-hover:underline decoration-dotted",
            selectedLog && selectedLog.request_id !== log.request_id && "pointer-events-none",
          )}
        />
      );
    },
  },
];
