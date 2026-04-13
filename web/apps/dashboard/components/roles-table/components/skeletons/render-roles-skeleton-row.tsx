import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { cn } from "@/lib/utils";
import type { DataTableColumnDef } from "@unkey/ui";
import {
  ActionColumnSkeleton,
  LastUpdatedColumnSkeleton,
  PermissionsColumnSkeleton,
} from "@unkey/ui";
import { ROLE_COLUMN_IDS } from "../../columns/create-roles-columns";
import { AssignedKeysColumnSkeleton } from "./assigned-keys-column-skeleton";
import { RoleColumnSkeleton } from "./role-column-skeleton";

type RenderRolesSkeletonRowProps = {
  columns: DataTableColumnDef<RoleBasic>[];
  rowHeight: number;
};

export const renderRolesSkeletonRow = ({ columns }: RenderRolesSkeletonRowProps) =>
  columns.map((column) => (
    <td
      key={column.id}
      className={cn("text-xs align-middle whitespace-nowrap", column.meta?.cellClassName)}
    >
      {column.id === ROLE_COLUMN_IDS.ROLE.id && <RoleColumnSkeleton />}
      {column.id === ROLE_COLUMN_IDS.ASSIGNED_KEYS.id && <AssignedKeysColumnSkeleton />}
      {column.id === ROLE_COLUMN_IDS.PERMISSIONS.id && <PermissionsColumnSkeleton />}
      {column.id === ROLE_COLUMN_IDS.LAST_UPDATED.id && <LastUpdatedColumnSkeleton />}
      {column.id === ROLE_COLUMN_IDS.ACTION.id && <ActionColumnSkeleton />}
    </td>
  ));
