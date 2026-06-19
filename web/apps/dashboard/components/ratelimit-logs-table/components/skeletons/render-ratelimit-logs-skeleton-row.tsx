import { cn } from "@/lib/utils";
import type { DataTableColumnDef } from "@unkey/ui";
import { CreatedAtColumnSkeleton } from "@unkey/ui";
import { RATELIMIT_LOGS_COLUMN_IDS } from "../../columns/create-ratelimit-logs-columns";
import type { EnrichedRatelimitLog } from "../../hooks/use-ratelimit-logs-query";

type RenderRatelimitLogsSkeletonRowProps = {
  columns: DataTableColumnDef<EnrichedRatelimitLog>[];
  rowHeight: number;
};

const TextSkeleton = ({ width }: { width: string }) => (
  <div className="h-3 rounded bg-grayA-3 animate-pulse" style={{ width }} />
);

// Matches the status badge's min-w-17.5 footprint so the skeleton doesn't reflow
// when real rows load in.
const BadgeSkeleton = () => <div className="h-4 w-17.5 rounded-md bg-grayA-3 animate-pulse" />;

export const renderRatelimitLogsSkeletonRow = ({
  columns,
  rowHeight,
}: RenderRatelimitLogsSkeletonRowProps) =>
  columns.map((column) => (
    <td
      key={column.id}
      className={cn("text-xs align-middle whitespace-nowrap", column.meta?.cellClassName)}
      style={{ height: `${rowHeight}px` }}
    >
      {column.id === RATELIMIT_LOGS_COLUMN_IDS.TIME && (
        <div className="pl-2">
          <CreatedAtColumnSkeleton />
        </div>
      )}
      {column.id === RATELIMIT_LOGS_COLUMN_IDS.IDENTIFIER && <TextSkeleton width="70%" />}
      {column.id === RATELIMIT_LOGS_COLUMN_IDS.STATUS && <BadgeSkeleton />}
      {column.id === RATELIMIT_LOGS_COLUMN_IDS.LIMIT && <TextSkeleton width="40%" />}
      {column.id === RATELIMIT_LOGS_COLUMN_IDS.DURATION && <TextSkeleton width="40%" />}
      {column.id === RATELIMIT_LOGS_COLUMN_IDS.RESET && <TextSkeleton width="60%" />}
      {column.id === RATELIMIT_LOGS_COLUMN_IDS.REGION && <TextSkeleton width="50%" />}
      {column.id === RATELIMIT_LOGS_COLUMN_IDS.ACTIONS && <TextSkeleton width="1rem" />}
    </td>
  ));
