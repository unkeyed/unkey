import { cn } from "@/lib/utils";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import type { DataTableColumnDef } from "@unkey/ui";
import { RegionCell, TagsCell, TimestampInfo } from "@unkey/ui";
import { OutcomeCell } from "../components/outcome-cell";

export const createKeyDetailsLogsColumns = (): DataTableColumnDef<KeyDetailsLog>[] => [
  {
    id: "time",
    accessorKey: "time",
    header: "Time",
    enableSorting: false,
    meta: {
      width: "20%",
      headerClassName: "pl-2",
      cellClassName: "pl-2",
    },
    cell: ({ row, table }) => (
      <TimestampInfo
        value={row.original.time}
        className={cn(
          "font-mono group-hover:underline decoration-dotted",
          table.getIsSomeRowsSelected() && !row.getIsSelected() && "pointer-events-none",
        )}
      />
    ),
  },
  {
    id: "outcome",
    accessorKey: "outcome",
    header: "Status",
    enableSorting: false,
    meta: {
      width: "20%",
    },
    cell: ({ row }) => {
      return <OutcomeCell log={row.original} isSelected={row.getIsSelected()} />;
    },
  },
  {
    id: "region",
    accessorKey: "region",
    header: "Region",
    enableSorting: false,
    meta: {
      width: "20%",
    },
    cell: ({ row }) => <RegionCell region={row.original.region} />,
  },
  {
    id: "tags",
    accessorKey: "tags",
    header: "Tags",
    enableSorting: false,
    meta: {
      width: "40%",
    },
    cell: ({ row }) => {
      return <TagsCell tags={row.original.tags ?? []} isSelected={row.getIsSelected()} />;
    },
  },
];
