import { RegionFlag } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/region-flag";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
import { cn } from "@/lib/utils";
import { Badge, type DataTableColumnDef, TimestampInfo } from "@unkey/ui";
import { type RuntimeLogRow, getLogKey, getSeverityStyle } from "../utils/get-row-class";

type CreateRuntimeLogsColumnsOptions = {
  // Key of the currently selected log, or null. Drives the selected badge
  // variant and disables timestamp hover on non-selected rows.
  selectedLogKey: string | null;
};

export const createRuntimeLogsColumns = ({
  selectedLogKey,
}: CreateRuntimeLogsColumnsOptions): DataTableColumnDef<RuntimeLogRow>[] => [
  {
    id: "time",
    accessorKey: "time",
    header: "Time",
    enableSorting: false,
    meta: {
      width: "12%",
      headerClassName: "pl-4",
    },
    cell: ({ row }) => (
      <div className="pl-4">
        <TimestampInfo
          value={row.original.time}
          className={cn(
            "font-mono group-hover:underline decoration-dotted",
            selectedLogKey && selectedLogKey !== getLogKey(row.original) && "pointer-events-none",
          )}
        />
      </div>
    ),
  },
  {
    id: "severity",
    accessorKey: "severity",
    header: "Severity",
    enableSorting: false,
    meta: {
      width: "10%",
    },
    cell: ({ row }) => {
      const style = getSeverityStyle(row.original.severity);
      const isSelected = selectedLogKey === getLogKey(row.original);
      return (
        <Badge
          className={cn(
            "uppercase px-1.5 rounded-md font-mono whitespace-nowrap",
            isSelected ? style.badge.selected : style.badge.default,
          )}
        >
          {row.original.severity}
        </Badge>
      );
    },
  },
  {
    id: "region",
    accessorKey: "region",
    header: "Region",
    enableSorting: false,
    meta: {
      width: "10%",
    },
    cell: ({ row }) => (
      <div className="flex items-center gap-1.5">
        <RegionFlag flagCode={mapRegionToFlag(row.original.region)} size="xs" shape="circle" />
        <span className="font-mono text-xs text-gray-11">{row.original.region}</span>
      </div>
    ),
  },
  {
    id: "instanceId",
    accessorKey: "instance_id",
    header: "Instance ID",
    enableSorting: false,
    meta: {
      width: "12%",
    },
    cell: ({ row }) => (
      <div className="font-mono pr-4 truncate text-xs" title={row.original.instance_id}>
        {row.original.instance_id}
      </div>
    ),
  },
  {
    id: "message",
    accessorKey: "message",
    header: "Message",
    enableSorting: false,
    meta: {
      width: "20%",
    },
    cell: ({ row }) => (
      <div className="font-mono truncate pr-4 max-w-75" title={row.original.message}>
        {row.original.message}
      </div>
    ),
  },
  {
    id: "attributes",
    accessorKey: "attributes",
    header: "Attributes",
    enableSorting: false,
    meta: {
      width: "auto",
    },
    cell: ({ row }) => {
      let attrStr = row.original.attributes ? JSON.stringify(row.original.attributes) : "";
      if (attrStr === "" || attrStr === "{}") {
        attrStr = "—";
      }
      return (
        <div className="font-mono truncate text-xs max-w-125" title={attrStr}>
          {attrStr}
        </div>
      );
    },
  },
];
