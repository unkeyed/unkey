import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { cn } from "@/lib/utils";
import type { DataTableColumnDef } from "@unkey/ui";
import { ActionColumnSkeleton, LastUpdatedColumnSkeleton } from "@unkey/ui";
import { PERMISSION_COLUMN_IDS } from "../../columns/create-permissions-columns";
import { AssignedKeysColumnSkeleton } from "./assigned-keys-column-skeleton";
import { AssignedRolesColumnSkeleton } from "./assigned-roles-column-skeleton";
import { PermissionColumnSkeleton } from "./permission-column-skeleton";
import { SlugColumnSkeleton } from "./slug-column-skeleton";

type RenderPermissionsSkeletonRowProps = {
  columns: DataTableColumnDef<Permission>[];
};

export const renderPermissionsSkeletonRow = ({ columns }: RenderPermissionsSkeletonRowProps) =>
  columns.map((column) => (
    <td
      key={column.id}
      className={cn("text-xs align-middle whitespace-nowrap", column.meta?.cellClassName)}
    >
      {column.id === PERMISSION_COLUMN_IDS.PERMISSION.id && <PermissionColumnSkeleton />}
      {column.id === PERMISSION_COLUMN_IDS.SLUG.id && <SlugColumnSkeleton />}
      {column.id === PERMISSION_COLUMN_IDS.USED_IN_ROLES.id && <AssignedRolesColumnSkeleton />}
      {column.id === PERMISSION_COLUMN_IDS.ASSIGNED_TO_KEYS.id && <AssignedKeysColumnSkeleton />}
      {column.id === PERMISSION_COLUMN_IDS.LAST_UPDATED.id && <LastUpdatedColumnSkeleton />}
      {column.id === PERMISSION_COLUMN_IDS.ACTION.id && <ActionColumnSkeleton />}
    </td>
  ));
