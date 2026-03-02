import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { cn } from "@/lib/utils";
import type { DataTableColumnDef } from "@unkey/ui";
import {
  ActionColumnSkeleton,
  CreatedAtColumnSkeleton,
  KeyColumnSkeleton,
  LastUpdatedColumnSkeleton,
  PermissionsColumnSkeleton,
  RootKeyColumnSkeleton,
} from "@unkey/ui";
import { ROOT_KEY_COLUMN_IDS } from "../../../root-keys-table/columns/create-root-key-columns";

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
