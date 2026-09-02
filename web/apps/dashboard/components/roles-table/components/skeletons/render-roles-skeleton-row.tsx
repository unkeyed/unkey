import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { cn } from "@/lib/utils";
import type { DataTableColumnDef } from "@unkey/ui";
import {
  ActionColumnSkeleton,
  DashedBadgeSkeleton,
  LastUpdatedColumnSkeleton,
  NameColumnSkeleton,
  PermissionsColumnSkeleton,
} from "@unkey/ui";
import { IconKey2Outline18, IconTagOutline18 } from "nucleo-ui-outline-18";
import { ROLE_COLUMN_IDS } from "../../columns/create-roles-columns";

type RenderRolesSkeletonRowProps = {
  columns: DataTableColumnDef<RoleBasic>[];
  rowHeight: number;
};

export const renderRolesSkeletonRow = ({ columns, rowHeight }: RenderRolesSkeletonRowProps) =>
  columns.map((column) => (
    <td
      key={column.id}
      className={cn("text-xs align-middle whitespace-nowrap", column.meta?.cellClassName)}
      style={{ height: `${rowHeight}px` }}
    >
      {column.id === ROLE_COLUMN_IDS.ROLE.id && (
        <NameColumnSkeleton icon={<IconTagOutline18 className="text-gray-12 opacity-50" />} />
      )}
      {column.id === ROLE_COLUMN_IDS.ASSIGNED_KEYS.id && (
        <DashedBadgeSkeleton
          icon={<IconKey2Outline18 className="size-4 opacity-50" />}
          barWidthClass="w-20"
          className="animate-in fade-in duration-300"
        />
      )}
      {column.id === ROLE_COLUMN_IDS.PERMISSIONS.id && <PermissionsColumnSkeleton />}
      {column.id === ROLE_COLUMN_IDS.LAST_UPDATED.id && <LastUpdatedColumnSkeleton />}
      {column.id === ROLE_COLUMN_IDS.ACTION.id && <ActionColumnSkeleton />}
    </td>
  ));
