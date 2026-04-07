import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import type { DataTableColumnDef } from "@unkey/ui";
import { LastUpdatedCell, RowActionSkeleton } from "@unkey/ui";
import dynamic from "next/dynamic";
import { AssignedItemsCell } from "../components/assigned-items-cell";
import { PermissionNameCell } from "../components/permission-name-cell";

const PermissionsTableActions = dynamic(
  () =>
    import("../components/actions/keys-table-action.popover.constants").then(
      (mod) => mod.PermissionsTableActions,
    ),
  {
    loading: () => <RowActionSkeleton />,
  },
);

export const PERMISSION_COLUMN_IDS = {
  PERMISSION: { id: "permission", accessorKey: "name", header: "Permission" },
  SLUG: { id: "slug", accessorKey: "slug", header: "Slug" },
  USED_IN_ROLES: {
    id: "used_in_roles",
    accessorKey: "totalConnectedRoles",
    header: "Used in Roles",
  },
  ASSIGNED_TO_KEYS: {
    id: "assigned_to_keys",
    accessorKey: "totalConnectedKeys",
    header: "Assigned to Keys",
  },
  LAST_UPDATED: { id: "last_updated", accessorKey: "lastUpdated", header: "Last Updated" },
  ACTION: { id: "action", accessorKey: "action", header: "" },
} as const;

type CreatePermissionsColumnsOptions = {
  selectedPermissionId?: string;
  selectedPermissions: Set<string>;
  hoveredPermissionName: string | null;
  onToggleSelection: (permissionId: string) => void;
  onHoverPermission: (name: string | null) => void;
};

export const createPermissionsColumns = ({
  selectedPermissionId,
  selectedPermissions,
  hoveredPermissionName,
  onToggleSelection,
  onHoverPermission,
}: CreatePermissionsColumnsOptions): DataTableColumnDef<Permission>[] => [
  {
    id: PERMISSION_COLUMN_IDS.PERMISSION.id,
    accessorKey: PERMISSION_COLUMN_IDS.PERMISSION.accessorKey,
    header: PERMISSION_COLUMN_IDS.PERMISSION.header,
    enableSorting: false,
    meta: {
      width: "20%",
      headerClassName: "pl-[18px]",
    },
    cell: ({ row }) => {
      const permission = row.original;
      return (
        <PermissionNameCell
          permission={permission}
          isChecked={selectedPermissions.has(permission.permissionId)}
          isHovered={hoveredPermissionName === permission.name}
          onToggleSelection={onToggleSelection}
          onHover={onHoverPermission}
        />
      );
    },
  },
  {
    id: PERMISSION_COLUMN_IDS.SLUG.id,
    accessorKey: PERMISSION_COLUMN_IDS.SLUG.accessorKey,
    header: PERMISSION_COLUMN_IDS.SLUG.header,
    enableSorting: false,
    meta: {
      width: "20%",
    },
    cell: ({ row }) => {
      const permission = row.original;
      return (
        <AssignedItemsCell
          kind="slug"
          value={permission.slug}
          isSelected={permission.permissionId === selectedPermissionId}
        />
      );
    },
  },
  {
    id: PERMISSION_COLUMN_IDS.USED_IN_ROLES.id,
    accessorKey: PERMISSION_COLUMN_IDS.USED_IN_ROLES.accessorKey,
    header: PERMISSION_COLUMN_IDS.USED_IN_ROLES.header,
    enableSorting: false,
    meta: {
      width: "20%",
    },
    cell: ({ row }) => {
      const permission = row.original;
      return (
        <AssignedItemsCell
          kind="roles"
          totalCount={permission.totalConnectedRoles}
          isSelected={permission.permissionId === selectedPermissionId}
        />
      );
    },
  },
  {
    id: PERMISSION_COLUMN_IDS.ASSIGNED_TO_KEYS.id,
    accessorKey: PERMISSION_COLUMN_IDS.ASSIGNED_TO_KEYS.accessorKey,
    header: PERMISSION_COLUMN_IDS.ASSIGNED_TO_KEYS.header,
    enableSorting: false,
    meta: {
      width: "20%",
    },
    cell: ({ row }) => {
      const permission = row.original;
      return (
        <AssignedItemsCell
          kind="keys"
          totalCount={permission.totalConnectedKeys}
          isSelected={permission.permissionId === selectedPermissionId}
        />
      );
    },
  },
  {
    id: PERMISSION_COLUMN_IDS.LAST_UPDATED.id,
    accessorKey: PERMISSION_COLUMN_IDS.LAST_UPDATED.accessorKey,
    header: PERMISSION_COLUMN_IDS.LAST_UPDATED.header,
    enableSorting: false,
    meta: {
      width: "12%",
    },
    cell: ({ row }) => {
      const permission = row.original;
      return (
        <LastUpdatedCell
          lastUpdated={permission.lastUpdated}
          isSelected={permission.permissionId === selectedPermissionId}
        />
      );
    },
  },
  {
    id: PERMISSION_COLUMN_IDS.ACTION.id,
    header: "",
    enableSorting: false,
    meta: {
      width: "auto",
    },
    cell: ({ row }) => {
      const permission = row.original;
      return <PermissionsTableActions permission={permission} />;
    },
  },
];
