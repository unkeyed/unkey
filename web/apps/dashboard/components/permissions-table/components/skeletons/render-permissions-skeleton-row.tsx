import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { cn } from "@/lib/utils";
import type { DataTableColumnDef } from "@unkey/ui";
import {
  ActionColumnSkeleton,
  DashedBadgeSkeleton,
  LastUpdatedColumnSkeleton,
  NameColumnSkeleton,
} from "@unkey/ui";
import { IconKey2Outline18, IconPage2Outline18, IconTagOutline18 } from "nucleo-ui-outline-18";
import { PERMISSION_COLUMN_IDS } from "../../columns/create-permissions-columns";

type RenderPermissionsSkeletonRowProps = {
  columns: DataTableColumnDef<Permission>[];
  rowHeight: number;
};

export const renderPermissionsSkeletonRow = ({
  columns,
  rowHeight,
}: RenderPermissionsSkeletonRowProps) =>
  columns.map((column) => (
    <td
      key={column.id}
      className={cn("text-xs align-middle whitespace-nowrap", column.meta?.cellClassName)}
      style={{ height: `${rowHeight}px` }}
    >
      {column.id === PERMISSION_COLUMN_IDS.PERMISSION.id && (
        <NameColumnSkeleton icon={<IconPage2Outline18 className="text-gray-12 opacity-50" />} />
      )}
      {column.id === PERMISSION_COLUMN_IDS.SLUG.id && (
        <DashedBadgeSkeleton icon={<IconPage2Outline18 className="size-4 opacity-50" />} />
      )}
      {column.id === PERMISSION_COLUMN_IDS.USED_IN_ROLES.id && (
        <DashedBadgeSkeleton icon={<IconTagOutline18 className="size-4 opacity-50" />} />
      )}
      {column.id === PERMISSION_COLUMN_IDS.ASSIGNED_TO_KEYS.id && (
        <DashedBadgeSkeleton icon={<IconKey2Outline18 className="size-4 opacity-50" />} />
      )}
      {column.id === PERMISSION_COLUMN_IDS.LAST_UPDATED.id && <LastUpdatedColumnSkeleton />}
      {column.id === PERMISSION_COLUMN_IDS.ACTION.id && <ActionColumnSkeleton />}
    </td>
  ));
