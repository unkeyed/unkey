import type { DataTableColumnDef } from "@/components/data-table";
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

export const renderRootKeySkeletonRow = ({
  columns,
  rowHeight,
}: RenderRootKeySkeletonRowProps) =>
  columns.map((column) => (
    <td
      key={column.id}
      className={cn(
        "text-xs align-middle whitespace-nowrap",
        column.id === "root_key" ? "py-[6px]" : "py-1",
        column.meta?.cellClassName,
      )}
      style={{ height: `${rowHeight}px` }}
    >
      {column.id === "root_key" && <RootKeyColumnSkeleton />}
      {column.id === "key" && <KeyColumnSkeleton />}
      {column.id === "created_at" && <CreatedAtColumnSkeleton />}
      {column.id === "permissions" && <PermissionsColumnSkeleton />}
      {column.id === "last_updated" && <LastUpdatedColumnSkeleton />}
      {column.id === "action" && <ActionColumnSkeleton />}
    </td>
  ));
