import { cn } from "@/lib/utils";
import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";
import { type DataTableColumnDef, RowActionSkeleton } from "@unkey/ui";

type RenderRatelimitsOverviewSkeletonRowProps = {
  columns: DataTableColumnDef<RatelimitOverviewLog>[];
  rowHeight: number;
};

const BadgeSkeleton = () => (
  <div className="h-5 w-16 rounded-md bg-grayA-3 animate-pulse" aria-hidden />
);

const IdentifierSkeleton = () => (
  <div className="flex items-center gap-3 pl-2">
    <div className="size-5 rounded-sm bg-grayA-3 animate-pulse" aria-hidden />
    <div className="h-4 w-40 rounded-sm bg-grayA-3 animate-pulse" aria-hidden />
  </div>
);

const TimestampSkeleton = () => (
  <div className="h-4 w-32 rounded-sm bg-grayA-3 animate-pulse" aria-hidden />
);

export const renderRatelimitsOverviewSkeletonRow = ({
  columns,
}: RenderRatelimitsOverviewSkeletonRowProps) =>
  columns.map((column) => (
    <td
      key={column.id}
      className={cn("text-xs align-middle whitespace-nowrap", column.meta?.cellClassName)}
    >
      {column.id === "identifier" && <IdentifierSkeleton />}
      {column.id === "passed" && <BadgeSkeleton />}
      {column.id === "blocked" && <BadgeSkeleton />}
      {column.id === "passed_tokens" && <BadgeSkeleton />}
      {column.id === "blocked_tokens" && <BadgeSkeleton />}
      {column.id === "time" && <TimestampSkeleton />}
      {column.id === "actions" && <RowActionSkeleton />}
    </td>
  ));
