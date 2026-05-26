import { cn } from "@/lib/utils";
import type { DataTableColumnDef } from "@unkey/ui";
import { SortableHeader, TimestampInfo } from "@unkey/ui";
import { LogsTableAction } from "../components/actions/logs-table-action";
import { DurationCell } from "../components/cells/duration-cell";
import { LimitCell } from "../components/cells/limit-cell";
import { RatelimitStatusCell } from "../components/cells/ratelimit-status-cell";
import { RegionValueCell } from "../components/cells/region-value-cell";
import { ResetCell } from "../components/cells/reset-cell";
import type { EnrichedRatelimitLog } from "../hooks/use-ratelimit-logs-query";

type CreateRatelimitLogsColumnsOptions = {
  // Sorting is server-side and only meaningful for historical pages, so the
  // live-tail view passes `false` to disable the sort affordance entirely.
  enableSorting?: boolean;
};

export const createRatelimitLogsColumns = ({
  enableSorting = false,
}: CreateRatelimitLogsColumnsOptions = {}): DataTableColumnDef<EnrichedRatelimitLog>[] => [
  {
    id: "time",
    accessorKey: "time",
    header: ({ header }) => <SortableHeader header={header}>Time</SortableHeader>,
    enableSorting,
    meta: {
      width: "10%",
      headerClassName: "pl-2",
    },
    cell: ({ row, table }) => {
      const pointerEventsNone = table.getIsSomeRowsSelected() && !row.getIsSelected();
      return (
        <div className="flex items-center gap-3 pl-2 truncate mr-3">
          <TimestampInfo
            value={row.original.time}
            className={cn(
              "font-mono group-hover:underline decoration-dotted",
              pointerEventsNone && "pointer-events-none",
            )}
          />
        </div>
      );
    },
  },
  {
    id: "identifier",
    accessorKey: "identifier",
    header: ({ header }) => <SortableHeader header={header}>Identifier</SortableHeader>,
    enableSorting,
    meta: {
      width: "15%",
    },
    cell: ({ row }) => (
      <div className="font-mono truncate mr-1 max-w-40">{row.original.identifier}</div>
    ),
  },
  {
    id: "status",
    accessorKey: "status",
    header: ({ header }) => <SortableHeader header={header}>Status</SortableHeader>,
    enableSorting,
    meta: {
      width: "10%",
    },
    cell: ({ row }) => <RatelimitStatusCell log={row.original} isSelected={row.getIsSelected()} />,
  },
  {
    id: "limit",
    header: "Limit",
    enableSorting: false,
    meta: {
      width: "auto",
    },
    cell: ({ row }) => <LimitCell log={row.original} />,
  },
  {
    id: "duration",
    header: "Duration",
    enableSorting: false,
    meta: {
      width: "auto",
    },
    cell: ({ row }) => <DurationCell log={row.original} />,
  },
  {
    id: "reset",
    header: "Resets At",
    enableSorting: false,
    meta: {
      width: "auto",
    },
    cell: ({ row, table }) => (
      <ResetCell
        log={row.original}
        pointerEventsNone={table.getIsSomeRowsSelected() && !row.getIsSelected()}
      />
    ),
  },
  {
    id: "region",
    accessorKey: "region",
    header: "Region",
    enableSorting: false,
    meta: {
      width: "10%",
    },
    cell: ({ row }) => <RegionValueCell log={row.original} />,
  },
  {
    id: "actions",
    header: "",
    enableSorting: false,
    meta: {
      width: "auto",
    },
    cell: ({ row }) => <LogsTableAction identifier={row.original.identifier} />,
  },
];
