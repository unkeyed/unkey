import type { RatelimitOverride } from "@/lib/collections";
import type { DataTableColumnDef } from "@unkey/ui";
import { ActionColumnSkeleton, Skeleton } from "@unkey/ui";
import { cn } from "cn";
import { OVERRIDE_COLUMN_IDS } from "../../columns/create-overrides-columns";

type RenderOverridesSkeletonRowProps = {
  columns: DataTableColumnDef<RatelimitOverride>[];
};

export const renderOverridesSkeletonRow = ({ columns }: RenderOverridesSkeletonRowProps) =>
  columns.map((column) => (
    <td
      key={column.id}
      className={cn("text-xs align-middle whitespace-nowrap", column.meta?.cellClassName)}
    >
      {column.id === OVERRIDE_COLUMN_IDS.ID && (
        <div className="pl-2">
          <Skeleton className="h-3 w-24 rounded" />
        </div>
      )}
      {column.id === OVERRIDE_COLUMN_IDS.IDENTIFIER && (
        <div className="pl-2">
          <Skeleton className="h-3 w-40 rounded" />
        </div>
      )}
      {column.id === OVERRIDE_COLUMN_IDS.LIMITS && (
        <Skeleton className="h-[22px] w-24 rounded-md" />
      )}
      {column.id === OVERRIDE_COLUMN_IDS.LAST_USED && (
        <Skeleton className="h-[22px] w-28 rounded-md" />
      )}
      {column.id === OVERRIDE_COLUMN_IDS.ACTIONS && <ActionColumnSkeleton />}
    </td>
  ));
