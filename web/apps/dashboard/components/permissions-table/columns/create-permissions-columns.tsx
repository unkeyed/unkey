import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import type { DataTableColumnDef } from "@unkey/ui";
import { LastUpdatedCell, RowActionSkeleton, SortableHeader } from "@unkey/ui";
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
    header: ({ header }) => (
      <SortableHeader header={header}>{PERMISSION_COLUMN_IDS.PERMISSION.header}</SortableHeader>
    ),
    enableSorting: true,
    meta: {
      width: {
        min: 260,
        max: 400,
      },
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
    header: ({ header }) => (
      <SortableHeader header={header}>{PERMISSION_COLUMN_IDS.SLUG.header}</SortableHeader>
    ),
    enableSorting: true,
    meta: {
      width: {
        min: 260,
        max: 400,
      },
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
    header: ({ header }) => (
      <SortableHeader header={header}>{PERMISSION_COLUMN_IDS.USED_IN_ROLES.header}</SortableHeader>
    ),
    enableSorting: true,
    meta: {
      width: {
        min: 200,
        max: 400,
      },
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
    header: ({ header }) => (
      <SortableHeader header={header}>
        {PERMISSION_COLUMN_IDS.ASSIGNED_TO_KEYS.header}
      </SortableHeader>
    ),
    enableSorting: true,
    meta: {
      width: {
        min: 200,
        max: 400,
      },
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
    sortDescFirst: true,
    header: ({ header }) => (
      <SortableHeader header={header}>{PERMISSION_COLUMN_IDS.LAST_UPDATED.header}</SortableHeader>
    ),
    enableSorting: true,
    meta: {
      width: {
        min: 200,
        max: 400,
      },
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
      width: {
        min: 60,
        max: 100,
      },
    },
    cell: ({ row }) => {
      const permission = row.original;
      return <PermissionsTableActions permission={permission} />;
    },
  },
];
