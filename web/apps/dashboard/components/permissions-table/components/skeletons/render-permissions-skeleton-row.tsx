import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { cn } from "@/lib/utils";
import { Key2, Page2, Tag } from "@unkey/icons";
import type { DataTableColumnDef } from "@unkey/ui";
import {
  ActionColumnSkeleton,
  DashedBadgeSkeleton,
  LastUpdatedColumnSkeleton,
  NameColumnSkeleton,
} from "@unkey/ui";
import { PERMISSION_COLUMN_IDS } from "../../columns/create-permissions-columns";

type RenderPermissionsSkeletonRowProps = {
  columns: DataTableColumnDef<Permission>[];
};

export const renderPermissionsSkeletonRow = ({ columns }: RenderPermissionsSkeletonRowProps) =>
  columns.map((column) => (
    <td
      key={column.id}
      className={cn("text-xs align-middle whitespace-nowrap", column.meta?.cellClassName)}
    >
      {column.id === PERMISSION_COLUMN_IDS.PERMISSION.id && (
        <NameColumnSkeleton
          icon={<Page2 iconSize="sm-regular" className="text-gray-12 opacity-50" />}
        />
      )}
      {column.id === PERMISSION_COLUMN_IDS.SLUG.id && (
        <DashedBadgeSkeleton icon={<Page2 iconSize="md-medium" className="opacity-50" />} />
      )}
      {column.id === PERMISSION_COLUMN_IDS.USED_IN_ROLES.id && (
        <DashedBadgeSkeleton icon={<Tag iconSize="md-medium" className="opacity-50" />} />
      )}
      {column.id === PERMISSION_COLUMN_IDS.ASSIGNED_TO_KEYS.id && (
        <DashedBadgeSkeleton icon={<Key2 iconSize="md-medium" className="opacity-50" />} />
      )}
      {column.id === PERMISSION_COLUMN_IDS.LAST_UPDATED.id && <LastUpdatedColumnSkeleton />}
      {column.id === PERMISSION_COLUMN_IDS.ACTION.id && <ActionColumnSkeleton />}
    </td>
  ));
