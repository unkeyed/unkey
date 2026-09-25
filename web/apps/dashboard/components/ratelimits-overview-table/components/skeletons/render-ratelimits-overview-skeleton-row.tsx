import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";
import { type DataTableColumnDef, RowActionSkeleton, Skeleton } from "@unkey/ui";
import { cn } from "cn";

type RenderRatelimitsOverviewSkeletonRowProps = {
  columns: DataTableColumnDef<RatelimitOverviewLog>[];
  rowHeight: number;
};

const BadgeSkeleton = () => <Skeleton className="h-5 w-16 rounded-md" />;

const IdentifierSkeleton = () => (
  <div className="flex items-center gap-3 pl-2">
    <Skeleton className="size-5" />
    <Skeleton className="h-4 w-40" />
  </div>
);

const TimestampSkeleton = () => <Skeleton className="h-4 w-32" />;

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
