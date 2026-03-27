import { cn } from "@/lib/utils";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import type { DataTableColumnDef } from "@unkey/ui";
import { RegionCell, TagsCell, TimestampInfo } from "@unkey/ui";
import { OutcomeCell } from "../components/outcome-cell";

type CreateColumnsOptions = {
  selectedLog: KeyDetailsLog | null;
};

export const createKeyDetailsLogsColumns = ({
  selectedLog,
}: CreateColumnsOptions): DataTableColumnDef<KeyDetailsLog>[] => [
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
    cell: ({ row }) => (
      <TimestampInfo
        value={row.original.time}
        className={cn(
          "font-mono group-hover:underline decoration-dotted",
          selectedLog &&
            selectedLog.request_id !== row.original.request_id &&
            "pointer-events-none",
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
      const log = row.original;
      const isSelected = selectedLog?.request_id === log.request_id;
      return <OutcomeCell log={log} isSelected={isSelected} />;
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
      const log = row.original;
      const isSelected = selectedLog?.request_id === log.request_id;
      return <TagsCell tags={log.tags ?? []} isSelected={isSelected} />;
    },
  },
];
