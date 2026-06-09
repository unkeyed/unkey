import { RegionFlag } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/region-flag";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
import { cn } from "@/lib/utils";
import { formatLatency } from "@/lib/utils/metric-formatters";
import type { SentinelLogsResponse } from "@unkey/clickhouse/src/sentinel";
import { TriangleWarning2 } from "@unkey/icons";
import { Badge, type DataTableColumnDef, TimestampInfo } from "@unkey/ui";
import { WARNING_ICON_STYLES, getStatusStyle } from "../utils/get-row-class";

const WarningIcon = ({ status }: { status: number }) => (
  <TriangleWarning2
    iconSize="md-regular"
    className={cn(
      WARNING_ICON_STYLES.base,
      status < 400 && "invisible",
      status >= 400 && status < 500 && WARNING_ICON_STYLES.warning,
      status >= 500 && WARNING_ICON_STYLES.error,
    )}
  />
);

export const createSentinelLogsColumns = (): DataTableColumnDef<SentinelLogsResponse>[] => [
  {
    id: "time",
    accessorKey: "time",
    header: "Time",
    enableSorting: false,
    meta: {
      width: "10%",
      headerClassName: "pl-8",
    },
    cell: ({ row }) => (
      <div className="flex items-center gap-3 px-2">
        <WarningIcon status={row.original.response_status} />
        <div className="flex-1 min-w-0">
          <TimestampInfo
            value={row.original.time}
            className="font-mono group-hover:underline decoration-dotted"
          />
        </div>
      </div>
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
    cell: ({ row }) => (
      <div className="flex items-center gap-1.5">
        <RegionFlag flagCode={mapRegionToFlag(row.original.region)} size="xs" shape="circle" />
        <span className="font-mono text-xs text-gray-11">{row.original.region}</span>
      </div>
    ),
  },
  {
    id: "response_status",
    accessorKey: "response_status",
    header: "Status",
    enableSorting: false,
    meta: {
      width: "5%",
    },
    cell: ({ row }) => {
      const style = getStatusStyle(row.original.response_status);
      return (
        <Badge
          className={cn(
            "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
            style.badge.default,
          )}
        >
          {row.original.response_status}
        </Badge>
      );
    },
  },
  {
    id: "method",
    accessorKey: "method",
    header: "Method",
    enableSorting: false,
    meta: {
      width: "5%",
    },
    cell: ({ row }) => (
      <Badge
        className={cn(
          "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
          getStatusStyle(row.original.response_status).badge.default,
        )}
      >
        {row.original.method}
      </Badge>
    ),
  },
  {
    id: "host",
    accessorKey: "host",
    header: "Host",
    enableSorting: false,
    meta: {
      width: "25%",
    },
    cell: ({ row }) => (
      <div className="font-mono pr-4 truncate" title={row.original.host}>
        {row.original.host}
      </div>
    ),
  },
  {
    id: "path",
    accessorKey: "path",
    header: "Path",
    enableSorting: false,
    meta: {
      width: "25%",
    },
    cell: ({ row }) => (
      <div className="font-mono pr-4 truncate max-w-[250px]" title={row.original.path}>
        {row.original.path}
      </div>
    ),
  },
  {
    id: "total_latency",
    accessorKey: "total_latency",
    header: "Total Latency",
    enableSorting: false,
    meta: {
      width: "10%",
    },
    cell: ({ row }) => (
      <div className="font-mono pr-4">{formatLatency(row.original.total_latency)}</div>
    ),
  },
];
