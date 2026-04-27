import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { Key2, Page2, Tag } from "@unkey/icons";
import type { DataTableColumnDef } from "@unkey/ui";
import {
  AssignedCountCell,
  LastUpdatedCell,
  RowActionSkeleton,
  SelectableNameCell,
  SortableHeader,
} from "@unkey/ui";
import dynamic from "next/dynamic";
import { SlugCell } from "../components/slug-cell";

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
      <div className="pl-5">
        <SortableHeader header={header}>{PERMISSION_COLUMN_IDS.PERMISSION.header}</SortableHeader>
      </div>
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
        <SelectableNameCell
          name={permission.name}
          description={permission.description}
          icon={<Page2 iconSize="sm-regular" className="text-gray-12 cursor-pointer" />}
          isSelected={selectedPermissions.has(permission.permissionId)}
          isHovered={hoveredPermissionName === permission.name}
          onMouseEnter={() => onHoverPermission(permission.name)}
          onMouseLeave={() => onHoverPermission(null)}
          onCheckedChange={() => onToggleSelection(permission.permissionId)}
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
        <SlugCell
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
    sortDescFirst: true,
    meta: {
      width: {
        min: 200,
        max: 400,
      },
    },
    cell: ({ row }) => {
      const permission = row.original;
      return (
        <AssignedCountCell
          count={permission.totalConnectedRoles}
          icon={<Tag iconSize="md-medium" className="opacity-50" />}
          singularLabel="Role"
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
    sortDescFirst: true,
    meta: {
      width: {
        min: 200,
        max: 400,
      },
    },
    cell: ({ row }) => {
      const permission = row.original;
      return (
        <AssignedCountCell
          count={permission.totalConnectedKeys}
          icon={<Key2 iconSize="md-medium" className="opacity-50" />}
          singularLabel="Key"
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
