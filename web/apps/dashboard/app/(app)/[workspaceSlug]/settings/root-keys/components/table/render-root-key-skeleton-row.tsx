import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { cn } from "@/lib/utils";
import { Dots } from "@unkey/icons";
import { type DataTableColumnDef, Skeleton } from "@unkey/ui";
import { ROOT_KEY_COLUMN_IDS } from "./create-root-key-columns";

const NameSkeleton = () => (
  <div className="px-[18px] py-[6px]">
    <Skeleton className="h-3.5 w-32" />
  </div>
);

const PermissionsSkeleton = () => (
  <div className="flex items-center gap-2">
    <Skeleton className="h-3.5 w-36" />
    <Skeleton className="h-3 w-5" />
  </div>
);

const CreatedAtSkeleton = () => <Skeleton className="h-3.5 w-24" />;

const ActionSkeleton = () => (
  <Skeleton className="flex size-7 items-center justify-center rounded-md border border-gray-6 bg-transparent">
    <Dots className="text-gray-11" iconSize="sm-regular" />
  </Skeleton>
);

type RenderRootKeySkeletonRowProps = {
  columns: DataTableColumnDef<RootKey>[];
  rowHeight: number;
};

export const renderRootKeySkeletonRow = ({ columns, rowHeight }: RenderRootKeySkeletonRowProps) =>
  columns.map((column) => (
    <td
      key={column.id}
      className={cn("text-xs align-middle whitespace-nowrap", column.meta?.cellClassName)}
      style={{ height: `${rowHeight}px` }}
    >
      {column.id === ROOT_KEY_COLUMN_IDS.ROOT_KEY.id && <NameSkeleton />}
      {column.id === ROOT_KEY_COLUMN_IDS.PERMISSIONS.id && <PermissionsSkeleton />}
      {column.id === ROOT_KEY_COLUMN_IDS.CREATED_AT.id && <CreatedAtSkeleton />}
      {column.id === ROOT_KEY_COLUMN_IDS.ACTION.id && <ActionSkeleton />}
    </td>
  ));
