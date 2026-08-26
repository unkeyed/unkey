import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { cn } from "@/lib/utils";
import { Dots } from "@unkey/icons";
import type { DataTableColumnDef } from "@unkey/ui";
import { ROOT_KEY_COLUMN_IDS } from "./create-root-key-columns";

const NameSkeleton = () => (
  <div className="px-[18px] py-[6px]">
    <div className="h-3.5 w-32 animate-pulse rounded-sm bg-grayA-3" />
  </div>
);

const PermissionsSkeleton = () => (
  <div className="flex items-center gap-2">
    <div className="h-3.5 w-36 animate-pulse rounded-sm bg-grayA-3" />
    <div className="h-3 w-5 animate-pulse rounded-sm bg-grayA-3" />
  </div>
);

const CreatedAtSkeleton = () => <div className="h-3.5 w-24 animate-pulse rounded-sm bg-grayA-3" />;

const ActionSkeleton = () => (
  <div className="flex size-7 animate-pulse items-center justify-center rounded-md border border-gray-6">
    <Dots className="text-gray-11" iconSize="sm-regular" />
  </div>
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
