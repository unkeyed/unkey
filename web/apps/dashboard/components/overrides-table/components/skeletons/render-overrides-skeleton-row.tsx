import type { RatelimitOverride } from "@/lib/collections";
import { cn } from "@/lib/utils";
import type { DataTableColumnDef } from "@unkey/ui";
import { ActionColumnSkeleton } from "@unkey/ui";
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
          <div className="h-3 w-24 bg-grayA-3 rounded animate-pulse" />
        </div>
      )}
      {column.id === OVERRIDE_COLUMN_IDS.IDENTIFIER && (
        <div className="pl-2">
          <div className="h-3 w-40 bg-grayA-3 rounded animate-pulse" />
        </div>
      )}
      {column.id === OVERRIDE_COLUMN_IDS.LIMITS && (
        <div className="h-[22px] w-24 bg-grayA-3 rounded-md animate-pulse" />
      )}
      {column.id === OVERRIDE_COLUMN_IDS.LAST_USED && (
        <div className="h-[22px] w-28 bg-grayA-3 rounded-md animate-pulse" />
      )}
      {column.id === OVERRIDE_COLUMN_IDS.ACTIONS && <ActionColumnSkeleton />}
    </td>
  ));
