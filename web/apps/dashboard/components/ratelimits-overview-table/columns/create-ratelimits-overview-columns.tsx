"use client";

import { LogsTableAction } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/_overview/components/table/components/logs-actions";
import { formatNumber } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";
import { Ban } from "@unkey/icons";
import { Badge, type DataTableColumnDef, SortableHeader, TimestampInfo } from "@unkey/ui";
import { IdentifierColumn } from "../components/identifier-column";
import { InlineFilter } from "../components/inline-filter";
import { STATUS_STYLES, getStatusStyle } from "../utils/get-row-class";

type CreateColumnsOptions = {
  namespaceId: string;
};

export const createRatelimitsOverviewColumns = ({
  namespaceId,
}: CreateColumnsOptions): DataTableColumnDef<RatelimitOverviewLog>[] => [
  {
    id: "identifier",
    accessorKey: "identifier",
    header: "Identifier",
    enableSorting: false,
    meta: {
      width: "18%",
      headerClassName: "pl-12",
    },
    cell: ({ row }) => (
      <div className="flex gap-3 items-center group/identifier min-w-0">
        <IdentifierColumn log={row.original} />
        <div className="shrink-0">
          <InlineFilter
            content="Filter by identifier"
            filterPair={{ identifiers: row.original.identifier }}
          />
        </div>
      </div>
    ),
  },
  {
    id: "passed",
    accessorKey: "passed_count",
    sortDescFirst: true,
    header: ({ header }) => <SortableHeader header={header}>Passed Requests</SortableHeader>,
    meta: {
      width: "15%",
    },
    cell: ({ row }) => {
      const log = row.original;
      const isSelected = row.getIsSelected();
      return (
        <div className="flex gap-3 items-center group/identifier">
          <Badge
            className={cn(
              "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
              isSelected
                ? STATUS_STYLES.success.badge.selected
                : STATUS_STYLES.success.badge.default,
            )}
            title={`${log.passed_count.toLocaleString()} Passed requests`}
          >
            {formatNumber(log.passed_count)}
          </Badge>
          <InlineFilter
            filterPair={{ identifiers: log.identifier, status: "passed" }}
            content="Filter by identifier and passed status"
          />
        </div>
      );
    },
  },
  {
    id: "blocked",
    accessorKey: "blocked_count",
    sortDescFirst: true,
    header: ({ header }) => <SortableHeader header={header}>Blocked Requests</SortableHeader>,
    meta: {
      width: "15%",
    },
    cell: ({ row }) => {
      const log = row.original;
      const isSelected = row.getIsSelected();
      const style = getStatusStyle(log);
      return (
        <div className="flex gap-3 items-center group/identifier">
          <Badge
            className={cn(
              "uppercase px-[6px] rounded-md font-mono whitespace-nowrap gap-[6px]",
              isSelected ? style.badge.selected : style.badge.default,
            )}
            title={`${log.blocked_count.toLocaleString()} Blocked requests`}
          >
            <Ban iconSize="sm-regular" />
            {formatNumber(log.blocked_count)}
          </Badge>
          <InlineFilter
            content="Filter by identifier and blocked status"
            filterPair={{ identifiers: log.identifier, status: "blocked" }}
          />
        </div>
      );
    },
  },
  {
    id: "passed_tokens",
    accessorKey: "passed_tokens",
    sortDescFirst: true,
    header: ({ header }) => <SortableHeader header={header}>Passed Tokens</SortableHeader>,
    meta: {
      width: "15%",
    },
    cell: ({ row }) => {
      const log = row.original;
      const isSelected = row.getIsSelected();
      return (
        <Badge
          className={cn(
            "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
            isSelected ? STATUS_STYLES.success.badge.selected : STATUS_STYLES.success.badge.default,
          )}
          title={`${log.passed_tokens.toLocaleString()} passed tokens`}
        >
          {formatNumber(log.passed_tokens)}
        </Badge>
      );
    },
  },
  {
    id: "blocked_tokens",
    accessorKey: "total_tokens",
    sortDescFirst: true,
    header: ({ header }) => <SortableHeader header={header}>Blocked Tokens</SortableHeader>,
    meta: {
      width: "15%",
    },
    cell: ({ row }) => {
      const log = row.original;
      const isSelected = row.getIsSelected();
      const blockedTokens = Math.max(log.total_tokens - log.passed_tokens, 0);
      const style = blockedTokens > 0 ? STATUS_STYLES.blocked : STATUS_STYLES.success;
      return (
        <Badge
          className={cn(
            "uppercase px-[6px] rounded-md font-mono whitespace-nowrap gap-[6px]",
            isSelected ? style.badge.selected : style.badge.default,
          )}
          title={`${blockedTokens.toLocaleString()} blocked tokens`}
        >
          <Ban iconSize="sm-regular" />
          {formatNumber(blockedTokens)}
        </Badge>
      );
    },
  },
  {
    id: "time",
    accessorKey: "time",
    sortDescFirst: true,
    header: ({ header }) => <SortableHeader header={header}>Last Request</SortableHeader>,
    meta: {
      width: "18%",
    },
    cell: ({ row, table }) => {
      const log = row.original;
      return (
        <TimestampInfo
          value={log.time}
          className={cn(
            "font-mono group-hover:underline decoration-dotted",
            table.getIsSomeRowsSelected() && !row.getIsSelected() && "pointer-events-none",
          )}
        />
      );
    },
  },
  {
    id: "actions",
    header: "",
    enableSorting: false,
    meta: {
      width: "auto",
    },
    cell: ({ row }) => (
      <LogsTableAction
        overrideDetails={row.original.override}
        identifier={row.original.identifier}
        namespaceId={namespaceId}
      />
    ),
  },
];
