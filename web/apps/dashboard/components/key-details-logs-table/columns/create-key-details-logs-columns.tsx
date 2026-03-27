import { cn } from "@/lib/utils";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import type { DataTableColumnDef } from "@unkey/ui";
import { InfoTooltip, TimestampInfo } from "@unkey/ui";
import { StatusBadge } from "../components/status-badge";
import { TagsCell } from "../components/tags-cell";
import {
  LOG_OUTCOME_DEFINITIONS,
  getStatusType,
  type LogOutcomeType,
} from "../utils/outcome-definitions";
import { STATUS_STYLES } from "../utils/get-row-class";

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
      const outcomeType =
        (log.outcome as LogOutcomeType) in LOG_OUTCOME_DEFINITIONS
          ? (log.outcome as LogOutcomeType)
          : "";
      const outcomeInfo = LOG_OUTCOME_DEFINITIONS[outcomeType];
      return (
        <InfoTooltip
          variant="inverted"
          className="cursor-default"
          content={<p>{outcomeInfo.tooltip}</p>}
          position={{ side: "top", align: "center", sideOffset: 5 }}
        >
          <div className="flex gap-3 items-center">
            <StatusBadge
              primary={{
                label: outcomeInfo.label,
                color: isSelected
                  ? STATUS_STYLES[getStatusType(outcomeInfo.type)].badge?.selected ?? ""
                  : STATUS_STYLES[getStatusType(outcomeInfo.type)].badge?.default ?? "",
                icon: outcomeInfo.icon,
              }}
            />
          </div>
        </InfoTooltip>
      );
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
    cell: ({ row }) => (
      <div className="flex items-center font-mono">
        <div className="w-full whitespace-nowrap" title={row.original.region}>
          {row.original.region}
        </div>
      </div>
    ),
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
      return <TagsCell log={log} isSelected={isSelected} />;
    },
  },
];
