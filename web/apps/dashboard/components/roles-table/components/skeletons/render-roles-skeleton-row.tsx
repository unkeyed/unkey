import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { cn } from "@/lib/utils";
import { Key2, Tag } from "@unkey/icons";
import type { DataTableColumnDef } from "@unkey/ui";
import {
  ActionColumnSkeleton,
  DashedBadgeSkeleton,
  LastUpdatedColumnSkeleton,
  NameColumnSkeleton,
  PermissionsColumnSkeleton,
} from "@unkey/ui";
import { ROLE_COLUMN_IDS } from "../../columns/create-roles-columns";

type RenderRolesSkeletonRowProps = {
  columns: DataTableColumnDef<RoleBasic>[];
};

export const renderRolesSkeletonRow = ({ columns }: RenderRolesSkeletonRowProps) =>
  columns.map((column) => (
    <td
      key={column.id}
      className={cn("text-xs align-middle whitespace-nowrap", column.meta?.cellClassName)}
    >
      {column.id === ROLE_COLUMN_IDS.ROLE.id && (
        <NameColumnSkeleton
          icon={<Tag iconSize="sm-regular" className="text-gray-12 opacity-50" />}
        />
      )}
      {column.id === ROLE_COLUMN_IDS.ASSIGNED_KEYS.id && (
        <DashedBadgeSkeleton
          icon={<Key2 iconSize="md-medium" className="opacity-50" />}
          barWidthClass="w-20"
          className="animate-in fade-in duration-300"
        />
      )}
      {column.id === ROLE_COLUMN_IDS.PERMISSIONS.id && <PermissionsColumnSkeleton />}
      {column.id === ROLE_COLUMN_IDS.LAST_UPDATED.id && <LastUpdatedColumnSkeleton />}
      {column.id === ROLE_COLUMN_IDS.ACTION.id && <ActionColumnSkeleton />}
    </td>
  ));
