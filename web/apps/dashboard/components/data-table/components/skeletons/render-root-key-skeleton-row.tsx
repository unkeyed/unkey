import type { DataTableColumnDef } from "@/components/data-table";
import { ROOT_KEY_COLUMN_IDS } from "@/components/data-table/columns/create-root-key-columns";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { cn } from "@/lib/utils";
import {
  ActionColumnSkeleton,
  CreatedAtColumnSkeleton,
  KeyColumnSkeleton,
  LastUpdatedColumnSkeleton,
  PermissionsColumnSkeleton,
  RootKeyColumnSkeleton,
} from "./root-key-skeletons";

type RenderRootKeySkeletonRowProps = {
  columns: DataTableColumnDef<RootKey>[];
  rowHeight: number;
};

export const renderRootKeySkeletonRow = ({ columns, rowHeight }: RenderRootKeySkeletonRowProps) =>
  columns.map((column) => (
    <td
      key={column.id}
      className={cn(
        "text-xs align-middle whitespace-nowrap",
        column.id === ROOT_KEY_COLUMN_IDS.ROOT_KEY ? "py-[6px]" : "py-1",
        column.meta?.cellClassName,
      )}
      style={{ height: `${rowHeight}px` }}
    >
      {column.id === ROOT_KEY_COLUMN_IDS.ROOT_KEY && <RootKeyColumnSkeleton />}
      {column.id === ROOT_KEY_COLUMN_IDS.KEY && <KeyColumnSkeleton />}
      {column.id === ROOT_KEY_COLUMN_IDS.CREATED_AT && <CreatedAtColumnSkeleton />}
      {column.id === ROOT_KEY_COLUMN_IDS.PERMISSIONS && <PermissionsColumnSkeleton />}
      {column.id === ROOT_KEY_COLUMN_IDS.LAST_UPDATED && <LastUpdatedColumnSkeleton />}
      {column.id === ROOT_KEY_COLUMN_IDS.ACTION && <ActionColumnSkeleton />}
    </td>
  ));
